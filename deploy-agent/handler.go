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

type deployResult struct {
	OK   bool   `json:"ok"`
	Tail string `json:"tail,omitempty"`
}

type deployHandler struct {
	secret      string
	mu          sync.Mutex
	inProgress  bool
	runPipeline func() (ok bool, tail string)
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
	ok, tail := h.runPipeline()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(deployResult{OK: ok, Tail: tail}); err != nil {
		log.Printf("deployHandler: encode response: %v", err)
	}
}
