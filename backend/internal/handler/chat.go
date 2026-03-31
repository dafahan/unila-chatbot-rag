package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/dafahan/unila-ai/internal/domain"
	"github.com/dafahan/unila-ai/internal/usecase"
)

type ChatHandler struct {
	uc *usecase.ChatUseCase
}

func NewChatHandler(uc *usecase.ChatUseCase) *ChatHandler {
	return &ChatHandler{uc: uc}
}

func (h *ChatHandler) Chat(w http.ResponseWriter, r *http.Request) {
	var req domain.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.Query) == "" {
		writeError(w, http.StatusBadRequest, "query is required")
		return
	}

	resp, err := h.uc.Answer(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// ChatStream handles the streaming chat endpoint using Server-Sent Events.
// Each SSE event carries a JSON payload:
//
//	token event : {"token": "..."}
//	done  event : {"done": true, "sources": [...]}
//	error event : {"error": "..."}
func (h *ChatHandler) ChatStream(w http.ResponseWriter, r *http.Request) {
	var req domain.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.Query) == "" {
		writeError(w, http.StatusBadRequest, "query is required")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable nginx buffering if present

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	sendEvent := func(v any) {
		data, _ := json.Marshal(v)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	sources, err := h.uc.AnswerStream(r.Context(), req, func(token string) {
		sendEvent(map[string]string{"token": token})
	})
	if err != nil {
		sendEvent(map[string]string{"error": err.Error()})
		return
	}

	// Final event: signal done and deliver sources
	type srcItem struct {
		Filename string `json:"filename"`
		Page     int    `json:"page_number"`
		Text     string `json:"text"`
	}
	items := make([]srcItem, 0, len(sources))
	for _, s := range sources {
		if s.Filename != "" {
			items = append(items, srcItem{Filename: s.Filename, Page: s.PageNumber, Text: s.Text})
		}
	}
	sendEvent(map[string]any{"done": true, "sources": items})
}
