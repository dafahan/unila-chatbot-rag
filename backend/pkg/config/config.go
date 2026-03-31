package config

import (
	"log"
	"os"
	"strconv"
)

type Config struct {
	Port      string
	UploadDir string

	LLMEngine string // "ollama" or "gemini"

	OllamaBaseURL    string
	OllamaModel      string
	OllamaEmbedModel string

	GeminiAPIKey    string
	GeminiModel     string
	GeminiEmbedModel string

	QdrantHost       string
	QdrantPort       int
	QdrantCollection string

	ChunkSize    int
	ChunkOverlap int
	TopK         int
}

func Load() *Config {
	return &Config{
		Port:      getEnv("PORT", "8080"),
		UploadDir: getEnv("UPLOAD_DIR", "./uploads"),

		LLMEngine: getEnv("LLM_ENGINE", "ollama"),

		OllamaBaseURL:    getEnv("OLLAMA_BASE_URL", "http://localhost:11434"),
		OllamaModel:      getEnv("OLLAMA_MODEL", "llama3:8b-instruct-q4_K_M"),
		OllamaEmbedModel: getEnv("OLLAMA_EMBED_MODEL", "nomic-embed-text"),

		GeminiAPIKey:     getEnv("GEMINI_API_KEY", ""),
		GeminiModel:      getEnv("GEMINI_MODEL", "gemini-1.5-flash"),
		GeminiEmbedModel: getEnv("GEMINI_EMBED_MODEL", "text-embedding-004"),

		QdrantHost:       getEnv("QDRANT_HOST", "localhost"),
		QdrantPort:       getEnvInt("QDRANT_PORT", 6334),
		QdrantCollection: getEnv("QDRANT_COLLECTION", "unila_docs"),

		ChunkSize:    getEnvInt("CHUNK_SIZE", 512),
		ChunkOverlap: getEnvInt("CHUNK_OVERLAP", 64),
		TopK:         getEnvInt("TOP_K", 5),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		log.Printf("warn: invalid int for %s, using default %d", key, fallback)
		return fallback
	}
	return i
}
