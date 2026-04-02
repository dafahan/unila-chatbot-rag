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

	// 4. Check context relevance — selects strict OOD guardrail mode when false.
	relevant := uc.checkRelevance(ctx, req.Query, chunks)

	// 5. Build prompt using the original query (user sees their own language)
	prompt := buildPrompt(req, chunks, relevant)

	// 6. Generate answer
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

// checkRelevance asks the LLM whether the retrieved chunks are relevant to the
// query. Returns true if context should be used, false if Qdrant likely
// returned off-topic results (hallucinated retrieval).
func (uc *ChatUseCase) checkRelevance(ctx context.Context, query string, chunks []domain.Chunk) bool {
	if len(chunks) == 0 {
		return false
	}

	var sb strings.Builder
	sb.WriteString("Kamu adalah penilai relevansi. Tugasmu HANYA menentukan apakah konteks berikut relevan untuk menjawab pertanyaan.\n\n")
	fmt.Fprintf(&sb, "Pertanyaan: %s\n\nKonteks:\n", query)
	for i, c := range chunks {
		fmt.Fprintf(&sb, "[%d] %s\n", i+1, c.Text)
	}
	sb.WriteString("\nApakah konteks di atas RELEVAN untuk menjawab pertanyaan tersebut? Jawab HANYA dengan satu kata: YA atau TIDAK.")

	result, err := uc.llm.GenerateCompletion(ctx, sb.String())
	if err != nil {
		return true // default: gunakan context jika gagal menilai
	}
	result = strings.ToLower(strings.TrimSpace(result))
	return strings.HasPrefix(result, "ya") || strings.HasPrefix(result, "yes")
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

	relevant := uc.checkRelevance(ctx, req.Query, chunks)
	prompt := buildPrompt(req, chunks, relevant)
	if err := uc.llm.GenerateCompletionStream(ctx, prompt, onToken); err != nil {
		return nil, fmt.Errorf("stream: %w", err)
	}
	return chunks, nil
}

var noInfoMessages = map[string]string{
	"id": "Informasi ini tidak tersedia pada dokumen regulasi UNILA. Silakan hubungi Admin UPT.",
	"en": "This information is not available in the UNILA regulatory documents. Please contact the UPT Admin.",
}

func noInfoMessage(lang string) string {
	if msg, ok := noInfoMessages[lang]; ok {
		return msg
	}
	return noInfoMessages["id"]
}

type promptRules struct {
	system      string
	commonRules []string
	// fallback when context is present but answer not found in it
	contextFallback string
	// noContextNote is injected when Qdrant result is not relevant
	noContextNote string
}

var langRules = map[string]promptRules{
	"id": {
		system: "Kamu adalah asisten akademik Universitas Lampung (UNILA).",
		commonRules: []string{
			"- Langsung ke isi jawaban, TANPA kalimat pembuka apapun.",
			"- DILARANG menyebut nama file, nama dokumen, atau nomor contoh dalam jawaban.",
			"- DILARANG menulis 'Menurut panduan...', 'Berdasarkan dokumen...', atau sejenisnya.",
			"- DILARANG menyebut gambar, foto, tabel, atau tangkapan layar — kamu tidak bisa menampilkannya.",
			"- Saat menjelaskan aksi di UI (klik, menu, tombol), SELALU sebutkan di menu atau halaman mana aksi tersebut berada.",
		},
		contextFallback: "- Jika informasi tidak ada dalam konteks, jawab HANYA: 'Informasi ini tidak tersedia pada dokumen regulasi UNILA. Silakan hubungi Admin UPT.'",
		noContextNote:   "CATATAN: Dokumen tidak memiliki informasi relevan. Jawab berdasarkan pengetahuanmu secara lengkap dan detail. Jika benar-benar tidak tahu, baru nyatakan informasi tidak tersedia.",
	},
	"en": {
		system: "You are an academic assistant for Universitas Lampung (UNILA).",
		commonRules: []string{
			"- Go straight to the answer, NO opening remarks whatsoever.",
			"- When describing a UI action (click, menu, button), ALWAYS state which menu or page it is located on.",
			"- NEVER mention file names, document names, or example numbers in your answer.",
			"- NEVER write 'According to the document...' or similar phrases.",
			"- NEVER reference images, figures, tables, or screenshots — you cannot display them.",
		},
		contextFallback: "- If information is not in the context, answer ONLY: 'This information is not available in the UNILA regulatory documents. Please contact the UPT Admin.'",
		noContextNote:   "NOTE: Documents have no relevant information. Answer fully and in detail from your own knowledge. Only state information is unavailable if you truly don't know.",
	},
}

func buildPrompt(req domain.ChatRequest, chunks []domain.Chunk, contextRelevant bool) string {
	lang := req.Language
	if lang != "en" {
		lang = "id"
	}
	r := langRules[lang]

	var sb strings.Builder
	fmt.Fprintf(&sb, "%s\nSTRICT RULES:\n", r.system)
	for _, rule := range r.commonRules {
		sb.WriteString(rule + "\n")
	}
	sb.WriteString(r.contextFallback + "\n")
	fmt.Fprintf(&sb, "- Use bullet points or numbering if listing items.\n- ALWAYS respond in the selected language (%s).\n\n", lang)

	if contextRelevant {
		sb.WriteString("=== CONTEXT ===\n")
		for i, c := range chunks {
			fmt.Fprintf(&sb, "[%d] (Source: %s, Page %d)\n%s\n\n", i+1, c.Filename, c.PageNumber, c.Text)
		}
		sb.WriteString("=== END CONTEXT ===\n\n")
	} else {
		sb.WriteString(r.noContextNote + "\n\n")
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
