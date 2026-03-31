package domain

import "context"

// LLMProvider is the core abstraction for all LLM interactions.
// Implementations: OllamaAdapter, GeminiAdapter.
type LLMProvider interface {
	GenerateCompletion(ctx context.Context, prompt string) (string, error)
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
	EmbeddingDimension() int
}
