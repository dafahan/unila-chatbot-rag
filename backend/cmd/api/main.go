package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"path/filepath"

	"github.com/joho/godotenv"

	"github.com/dafahan/unila-ai/internal/adapter"
	"github.com/dafahan/unila-ai/internal/domain"
	"github.com/dafahan/unila-ai/internal/handler"
	"github.com/dafahan/unila-ai/internal/repository"
	"github.com/dafahan/unila-ai/internal/usecase"
	"github.com/dafahan/unila-ai/pkg/bm25"
	"github.com/dafahan/unila-ai/pkg/config"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using environment variables")
	}
	cfg := config.Load()

	// --- Wire LLM Provider ---
	var llm domain.LLMProvider
	switch cfg.LLMEngine {
	case "gemini":
		g, err := adapter.NewGeminiAdapter(context.Background(), cfg.GeminiAPIKey, cfg.GeminiModel, cfg.GeminiEmbedModel)
		if err != nil {
			log.Fatalf("gemini adapter: %v", err)
		}
		llm = g
	default: // "ollama"
		o, err := adapter.NewOllamaAdapter(cfg.OllamaBaseURL, cfg.OllamaModel, cfg.OllamaEmbedModel)
		if err != nil {
			log.Fatalf("ollama adapter: %v", err)
		}
		llm = o
	}

	// --- Wire Repository ---
	repo, err := repository.NewQdrantRepository(cfg.QdrantHost, cfg.QdrantPort, cfg.QdrantCollection)
	if err != nil {
		log.Fatalf("qdrant: %v", err)
	}

	// --- Wire BM25 Index ---
	bm25Idx, err := bm25.Load(filepath.Join(cfg.UploadDir, ".bm25_stats.json"))
	if err != nil {
		log.Fatalf("bm25 index: %v", err)
	}

	// --- Wire Use Cases ---
	chatUC := usecase.NewChatUseCase(llm, repo, cfg, bm25Idx)
	ingestUC := usecase.NewIngestionUseCase(llm, repo, cfg, bm25Idx)

	// --- Wire Handlers ---
	chatH := handler.NewChatHandler(chatUC)
	docH := handler.NewDocumentHandler(ingestUC, repo, cfg.UploadDir)

	// --- Router ---
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/chat", chatH.Chat)
	mux.HandleFunc("POST /api/chat/stream", chatH.ChatStream)
	mux.HandleFunc("POST /api/documents/upload", docH.Upload)
	mux.HandleFunc("GET /api/documents", docH.List)
	mux.HandleFunc("DELETE /api/documents/{filename}", docH.Delete)
	mux.Handle("GET /uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir(cfg.UploadDir))))

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("server starting on %s (LLM engine: %s)", addr, cfg.LLMEngine)
	if err := http.ListenAndServe(addr, corsMiddleware(mux)); err != nil {
		log.Fatal(err)
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
