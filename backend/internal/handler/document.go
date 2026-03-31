package handler

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/dafahan/unila-ai/internal/domain"
	"github.com/dafahan/unila-ai/internal/usecase"
	pdfextract "github.com/dafahan/unila-ai/pkg/pdf"
)

type DocumentHandler struct {
	uc      *usecase.IngestionUseCase
	repo    domain.DocumentRepository
	uploadDir string
}

func NewDocumentHandler(uc *usecase.IngestionUseCase, repo domain.DocumentRepository, uploadDir string) *DocumentHandler {
	os.MkdirAll(uploadDir, 0755)
	return &DocumentHandler{uc: uc, repo: repo, uploadDir: uploadDir}
}

func (h *DocumentHandler) Upload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "file too large")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing file field")
		return
	}
	defer file.Close()

	raw, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "read file error")
		return
	}

	// Simpan PDF ke disk
	dst := filepath.Join(h.uploadDir, filepath.Base(header.Filename))
	if err := os.WriteFile(dst, raw, 0644); err != nil {
		writeError(w, http.StatusInternalServerError, "save file error")
		return
	}

	pages, err := pdfextract.ExtractPages(raw)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, fmt.Sprintf("pdf extract: %v", err))
		return
	}

	count, err := h.uc.IngestPages(r.Context(), header.Filename, pages)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"filename": header.Filename,
		"chunks":   count,
	})
}

func (h *DocumentHandler) List(w http.ResponseWriter, r *http.Request) {
	docs, err := h.repo.ListDocuments()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, docs)
}

func (h *DocumentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	filename := r.PathValue("filename")
	if filename == "" {
		writeError(w, http.StatusBadRequest, "filename is required")
		return
	}

	// Hapus dari Qdrant
	if _, err := h.repo.DeleteByFilename(filename); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Hapus file dari disk (ignore error jika tidak ada)
	os.Remove(filepath.Join(h.uploadDir, filepath.Base(filename)))

	writeJSON(w, http.StatusOK, map[string]string{"deleted": filename})
}
