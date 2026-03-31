package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/dafahan/unila-ai/internal/domain"
	"github.com/dafahan/unila-ai/pkg/config"
)

type ChatUseCase struct {
	llm  domain.LLMProvider
	repo domain.DocumentRepository
	cfg  *config.Config
}

func NewChatUseCase(llm domain.LLMProvider, repo domain.DocumentRepository, cfg *config.Config) *ChatUseCase {
	return &ChatUseCase{llm: llm, repo: repo, cfg: cfg}
}

func (uc *ChatUseCase) Answer(ctx context.Context, req domain.ChatRequest) (*domain.ChatResponse, error) {
	// 1. Embed the query
	queryVec, err := uc.llm.GenerateEmbedding(ctx, req.Query)
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}

	// 2. Retrieve top-K relevant chunks (hybrid: vector + keyword boost)
	keywords := extractKeywords(req.Query)
	chunks, err := uc.repo.SearchSimilar(queryVec, keywords, uc.cfg.TopK)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}

	// 3. Build prompt
	prompt := buildPrompt(req, chunks)

	// 4. Generate answer
	answer, err := uc.llm.GenerateCompletion(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("generate: %w", err)
	}

	return &domain.ChatResponse{
		Answer:  answer,
		Sources: chunks,
	}, nil
}

// extractKeywords returns meaningful words from query (skip stopwords).
func extractKeywords(query string) []string {
	stopwords := map[string]bool{
		"apa": true, "yang": true, "di": true, "ke": true, "dari": true,
		"dan": true, "untuk": true, "dengan": true, "ini": true, "itu": true,
		"adalah": true, "bagaimana": true, "cara": true, "tentang": true,
		"pada": true, "dalam": true, "atau": true, "juga": true, "ada": true,
		"tidak": true, "bisa": true, "saya": true, "kamu": true, "gua": true,
	}

	words := strings.Fields(strings.ToLower(query))
	var keywords []string
	for _, w := range words {
		if len(w) > 3 && !stopwords[w] {
			keywords = append(keywords, w)
		}
	}
	return keywords
}

var promptLang = map[string][6]string{
	"en": {
		"You are an academic assistant for Universitas Lampung (UNILA).",
		"Answer DIRECTLY and COMPLETELY based on the document context below.",
		"- Go straight to the answer, NO opening remarks whatsoever.",
		"- NEVER mention file names, document names, or example numbers in your answer.",
		"- NEVER write 'According to the document...' or similar phrases.",
		"- If information is unavailable, answer ONLY: 'This information is not available. Please contact the UPT Admin.'",
	},
	"id": {
		"Kamu adalah asisten akademik Universitas Lampung (UNILA).",
		"Jawab LANGSUNG dan LENGKAP berdasarkan konteks dokumen di bawah.",
		"- Langsung ke isi jawaban, TANPA kalimat pembuka apapun.",
		"- DILARANG menyebut nama file, nama dokumen, atau nomor contoh dalam jawaban.",
		"- DILARANG menulis 'Menurut panduan...', 'Berdasarkan dokumen...', atau sejenisnya.",
		"- Jika informasi tidak tersedia, jawab HANYA: 'Informasi ini tidak tersedia. Silakan hubungi Admin UPT.'",
	},
}

func buildPrompt(req domain.ChatRequest, chunks []domain.Chunk) string {
	lang := req.Language
	if lang != "id" {
		lang = "en" // default English
	}
	p := promptLang[lang]

	var sb strings.Builder
	fmt.Fprintf(&sb, "%s %s\nSTRICT RULES:\n%s\n%s\n%s\n%s\n- Use bullet points or numbering if listing items.\n- ALWAYS respond in the selected language (%s).\n\n",
		p[0], p[1], p[2], p[3], p[4], p[5], lang)

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
