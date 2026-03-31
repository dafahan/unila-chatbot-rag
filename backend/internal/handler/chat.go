package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/dafahan/unila-ai/internal/domain"
	"github.com/dafahan/unila-ai/internal/usecase"
)

type ChatHandler struct {
	uc *usecase.ChatUseCase
}

func NewChatHandler(uc *usecase.ChatUseCase) *ChatHandler {
	return &ChatHandler{uc: uc}
}

func (h *ChatHandler) Chat(w http.ResponseWriter, r *http.Request) {
	var req domain.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.Query) == "" {
		writeError(w, http.StatusBadRequest, "query is required")
		return
	}

	resp, err := h.uc.Answer(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}
