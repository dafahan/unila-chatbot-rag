package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/dafahan/unila-ai/internal/domain"
	"github.com/dafahan/unila-ai/pkg/bm25"
	"github.com/dafahan/unila-ai/pkg/config"
)

type ChatUseCase struct {
	llm  domain.LLMProvider
	repo domain.DocumentRepository
	cfg  *config.Config
	bm25 *bm25.Index
}

func NewChatUseCase(llm domain.LLMProvider, repo domain.DocumentRepository, cfg *config.Config, bm25Idx *bm25.Index) *ChatUseCase {
	return &ChatUseCase{llm: llm, repo: repo, cfg: cfg, bm25: bm25Idx}
}

func (uc *ChatUseCase) Answer(ctx context.Context, req domain.ChatRequest) (*domain.ChatResponse, error) {
	// For English queries, translate to Indonesian before retrieval so that
	// both dense embeddings and BM25 match against the Indonesian document corpus.
	// The original English query is still used in the prompt so the LLM answers in English.
	retrievalQuery := req.Query
	if req.Language == "en" {
		if translated, err := uc.translateToID(ctx, req.Query); err == nil && translated != "" {
			retrievalQuery = translated
		}
		// On failure, fall back to original query (dense-only retrieval, BM25 silent).
	}

	// 1. Embed the retrieval query (dense vector)
	queryVec, err := uc.llm.GenerateEmbedding(ctx, retrievalQuery)
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}

	// 2. Compute BM25 sparse query vector (lexical signal)
	sparseIdx, sparseVal := uc.bm25.VectorizeQuery(retrievalQuery)

	// 3. Hybrid retrieval: dense + sparse via Qdrant RRF fusion
	chunks, err := uc.repo.SearchSimilar(queryVec, sparseIdx, sparseVal, uc.cfg.TopK, float32(uc.cfg.ScoreThreshold))
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}

	// 4. Build prompt using the original query (user sees their own language)
	prompt := buildPrompt(req, chunks)

	// 5. Generate answer
	answer, err := uc.llm.GenerateCompletion(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("generate: %w", err)
	}

	return &domain.ChatResponse{
		Answer:  answer,
		Sources: chunks,
	}, nil
}

// translateToID translates an English query to Bahasa Indonesia using the
// configured LLM. The result is used only for retrieval — not shown to the user.
func (uc *ChatUseCase) translateToID(ctx context.Context, query string) (string, error) {
	prompt := "Translate the following question to Bahasa Indonesia. Return ONLY the translation, no explanation, no punctuation changes.\n\nQuestion: " + query + "\n\nTranslation:"
	result, err := uc.llm.GenerateCompletion(ctx, prompt)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(result), nil
}

// AnswerStream runs the full RAG pipeline and streams LLM tokens to onToken.
// Returns the retrieved source chunks after streaming completes.
func (uc *ChatUseCase) AnswerStream(ctx context.Context, req domain.ChatRequest, onToken func(string)) ([]domain.Chunk, error) {
	retrievalQuery := req.Query
	if req.Language == "en" {
		if translated, err := uc.translateToID(ctx, req.Query); err == nil && translated != "" {
			retrievalQuery = translated
		}
	}

	queryVec, err := uc.llm.GenerateEmbedding(ctx, retrievalQuery)
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}

	sparseIdx, sparseVal := uc.bm25.VectorizeQuery(retrievalQuery)
	chunks, err := uc.repo.SearchSimilar(queryVec, sparseIdx, sparseVal, uc.cfg.TopK, float32(uc.cfg.ScoreThreshold))
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}

	prompt := buildPrompt(req, chunks)
	if err := uc.llm.GenerateCompletionStream(ctx, prompt, onToken); err != nil {
		return nil, fmt.Errorf("stream: %w", err)
	}
	return chunks, nil
}

var promptLang = map[string][8]string{
	"en": {
		"You are an academic assistant for Universitas Lampung (UNILA).",
		"Answer DIRECTLY and COMPLETELY based on the document context below.",
		"- Go straight to the answer, NO opening remarks whatsoever.",
		"- When describing a UI action (click, menu, button), ALWAYS state which menu or page it is located on.",
		"- NEVER mention file names, document names, or example numbers in your answer.",
		"- NEVER write 'According to the document...' or similar phrases.",
		"- NEVER reference images, figures, tables, or screenshots (e.g. 'as shown in the image below') — you cannot display them.",
		"- If information is unavailable, answer ONLY: 'This information is not available. Please contact the UPT Admin.'",
	},
	"id": {
		"Kamu adalah asisten akademik Universitas Lampung (UNILA).",
		"Jawab LANGSUNG dan LENGKAP berdasarkan konteks dokumen di bawah.",
		"- Langsung ke isi jawaban, TANPA kalimat pembuka apapun.",
		"- DILARANG menyebut nama file, nama dokumen, atau nomor contoh dalam jawaban.",
		"- DILARANG menulis 'Menurut panduan...', 'Berdasarkan dokumen...', atau sejenisnya.",
		"- DILARANG menyebut gambar, foto, tabel, atau tangkapan layar (contoh: 'seperti gambar berikut') — kamu tidak bisa menampilkannya.",
		"- Saat menjelaskan aksi di UI (klik, menu, tombol), SELALU sebutkan di menu atau halaman mana aksi tersebut berada.",
		"- Jika informasi tidak tersedia, jawab HANYA: 'Informasi ini tidak tersedia. Silakan hubungi Admin UPT.'",
	},
}

func buildPrompt(req domain.ChatRequest, chunks []domain.Chunk) string {
	lang := req.Language
	if lang != "en" {
		lang = "id"
	}
	p := promptLang[lang]

	var sb strings.Builder
	fmt.Fprintf(&sb, "%s %s\nSTRICT RULES:\n%s\n%s\n%s\n%s\n%s\n%s\n- Use bullet points or numbering if listing items.\n- ALWAYS respond in the selected language (%s).\n\n",
		p[0], p[1], p[2], p[3], p[4], p[5], p[6], p[7], lang)

	if len(chunks) > 0 {
		sb.WriteString("=== CONTEXT ===\n")
		for i, c := range chunks {
			fmt.Fprintf(&sb, "[%d] (Source: %s, Page %d)\n%s\n\n", i+1, c.Filename, c.PageNumber, c.Text)
		}
		sb.WriteString("=== END CONTEXT ===\n\n")
	}

	if len(req.History) > 0 {
		sb.WriteString("=== CONVERSATION HISTORY ===\n")
		for _, msg := range req.History {
			fmt.Fprintf(&sb, "%s: %s\n", strings.ToUpper(msg.Role), msg.Content)
		}
		sb.WriteString("=== END HISTORY ===\n\n")
	}

	fmt.Fprintf(&sb, "STUDENT QUESTION: %s\n\nANSWER:", req.Query)
	return sb.String()
}
