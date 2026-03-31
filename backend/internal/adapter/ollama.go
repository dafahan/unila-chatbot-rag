package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type OllamaAdapter struct {
	baseURL    string
	model      string
	embedModel string
	dimension  int
	client     *http.Client
}

func NewOllamaAdapter(baseURL, model, embedModel string) *OllamaAdapter {
	return &OllamaAdapter{
		baseURL:    baseURL,
		model:      model,
		embedModel: embedModel,
		dimension:  768, // nomic-embed-text default
		client:     &http.Client{},
	}
}

func (o *OllamaAdapter) GenerateCompletion(ctx context.Context, prompt string) (string, error) {
	body, _ := json.Marshal(map[string]any{
		"model":  o.model,
		"prompt": prompt,
		"stream": false,
		"options": map[string]any{
			"temperature": 0.3,
			"top_p":       0.9,
		},
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.baseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ollama generate: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("ollama decode: %w", err)
	}
	return result.Response, nil
}

func (o *OllamaAdapter) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	body, _ := json.Marshal(map[string]any{
		"model":  o.embedModel,
		"prompt": text,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.baseURL+"/api/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama embed: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	var result struct {
		Embedding []float32 `json:"embedding"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("ollama embed decode: %w", err)
	}
	return result.Embedding, nil
}

func (o *OllamaAdapter) EmbeddingDimension() int {
	return o.dimension
}
