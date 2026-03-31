package adapter

import (
	"context"
	"fmt"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type GeminiAdapter struct {
	client     *genai.Client
	model      string
	embedModel string
	dimension  int
}

func NewGeminiAdapter(ctx context.Context, apiKey, model, embedModel string) (*GeminiAdapter, error) {
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("gemini client: %w", err)
	}
	return &GeminiAdapter{
		client:     client,
		model:      model,
		embedModel: embedModel,
		dimension:  768, // text-embedding-004 default
	}, nil
}

func (g *GeminiAdapter) GenerateCompletion(ctx context.Context, prompt string) (string, error) {
	m := g.client.GenerativeModel(g.model)
	resp, err := m.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("gemini generate: %w", err)
	}
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini: empty response")
	}
	part := resp.Candidates[0].Content.Parts[0]
	text, ok := part.(genai.Text)
	if !ok {
		return "", fmt.Errorf("gemini: unexpected part type")
	}
	return string(text), nil
}

func (g *GeminiAdapter) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	em := g.client.EmbeddingModel(g.embedModel)
	res, err := em.EmbedContent(ctx, genai.Text(text))
	if err != nil {
		return nil, fmt.Errorf("gemini embed: %w", err)
	}
	return res.Embedding.Values, nil
}

func (g *GeminiAdapter) EmbeddingDimension() int {
	return g.dimension
}
