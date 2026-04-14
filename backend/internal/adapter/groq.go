package adapter

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const groqBaseURL = "https://api.groq.com/openai/v1"

// GroqAdapter uses Groq for LLM generation (fast cloud inference)
// and Ollama for embeddings (bge-m3 running locally).
type GroqAdapter struct {
	groqKey      string
	model        string
	ollamaURL    string
	embedModel   string
	dimension    int
	httpClient   *http.Client
}

func NewGroqAdapter(ctx context.Context, groqKey, model, ollamaURL, embedModel string) (*GroqAdapter, error) {
	a := &GroqAdapter{
		groqKey:    groqKey,
		model:      model,
		ollamaURL:  ollamaURL,
		embedModel: embedModel,
		httpClient: &http.Client{},
	}
	vec, err := a.GenerateEmbedding(ctx, "test")
	if err != nil {
		return nil, fmt.Errorf("groq adapter: probe embedding dimension: %w", err)
	}
	a.dimension = len(vec)
	return a, nil
}

// --- Generation (Groq) ---

func (g *GroqAdapter) groqPost(ctx context.Context, prompt string, stream bool) (*http.Response, error) {
	body, _ := json.Marshal(map[string]any{
		"model": g.model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"stream":      stream,
		"temperature": 0.3,
		"top_p":       0.9,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, groqBaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+g.groqKey)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("groq: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("groq %d: %s", resp.StatusCode, raw)
	}
	return resp, nil
}

func (g *GroqAdapter) GenerateCompletion(ctx context.Context, prompt string) (string, error) {
	resp, err := g.groqPost(ctx, prompt, false)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("groq decode: %w", err)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("groq: empty response")
	}
	return result.Choices[0].Message.Content, nil
}

func (g *GroqAdapter) GenerateCompletionStream(ctx context.Context, prompt string, onToken func(string)) error {
	resp, err := g.groqPost(ctx, prompt, true)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			onToken(chunk.Choices[0].Delta.Content)
		}
	}
	return scanner.Err()
}

// --- Embedding (Ollama bge-m3) ---

func (g *GroqAdapter) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	body, _ := json.Marshal(map[string]any{
		"model":  g.embedModel,
		"prompt": text,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.ollamaURL+"/api/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("groq adapter embed (ollama): %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	var result struct {
		Embedding []float32 `json:"embedding"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("groq adapter embed decode: %w", err)
	}
	return result.Embedding, nil
}

func (g *GroqAdapter) EmbeddingDimension() int {
	return g.dimension
}
