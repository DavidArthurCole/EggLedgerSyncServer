package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
)

func tailLines(s string, n int) string {
	if s == "" || n <= 0 {
		return ""
	}
	lines := strings.Split(strings.TrimRight(s, "\r\n"), "\n")
	if len(lines) <= n {
		return strings.Join(lines, "\n")
	}
	return strings.Join(lines[len(lines)-n:], "\n")
}

type pipelineResult struct {
	OK             bool
	AlreadyUpToDate bool
	Tail           string
	FromHash       string
	ToHash         string
}

type deployResult struct {
	OK             bool   `json:"ok"`
	AlreadyUpToDate bool   `json:"alreadyUpToDate,omitempty"`
	Tail           string `json:"tail,omitempty"`
	FromHash       string `json:"fromHash,omitempty"`
	ToHash         string `json:"toHash,omitempty"`
}

type deployHandler struct {
	secret      string
	mu          sync.Mutex
	inProgress  bool
	runPipeline func() pipelineResult
}

func (h *deployHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if h.secret == "" || token != h.secret {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	h.mu.Lock()
	if h.inProgress {
		h.mu.Unlock()
		http.Error(w, "deploy already in progress", http.StatusConflict)
		return
	}
	h.inProgress = true
	h.mu.Unlock()
	defer func() {
		h.mu.Lock()
		h.inProgress = false
		h.mu.Unlock()
	}()
	result := h.runPipeline()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(deployResult{
		OK:              result.OK,
		AlreadyUpToDate: result.AlreadyUpToDate,
		Tail:            result.Tail,
		FromHash:        result.FromHash,
		ToHash:          result.ToHash,
	}); err != nil {
		log.Printf("deployHandler: encode response: %v", err)
	}
}
