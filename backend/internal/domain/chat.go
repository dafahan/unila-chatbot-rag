package domain

// ChatMessage represents a turn in a conversation.
type ChatMessage struct {
	Role    string `json:"role"` // "user" or "assistant"
	Content string `json:"content"`
}

// ChatRequest is the payload from the frontend.
type ChatRequest struct {
	Query    string        `json:"query"`
	History  []ChatMessage `json:"history,omitempty"`
	Language string        `json:"language,omitempty"` // "en" or "id", default "en"
}

// ChatResponse is returned to the frontend.
type ChatResponse struct {
	Answer  string  `json:"answer"`
	Sources []Chunk `json:"sources,omitempty"`
}
