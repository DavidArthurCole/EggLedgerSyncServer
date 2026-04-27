package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTailLines(t *testing.T) {
	tests := []struct {
		name  string
		input string
		n     int
		want  string
	}{
		{"empty", "", 5, ""},
		{"n is zero", "a\nb\nc", 0, ""},
		{"fewer than n", "a\nb\nc", 5, "a\nb\nc"},
		{"exact n", "a\nb\nc", 3, "a\nb\nc"},
		{"more than n", "a\nb\nc\nd\ne", 3, "c\nd\ne"},
		{"trailing newline", "a\nb\nc\n", 2, "b\nc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tailLines(tt.input, tt.n)
			if got != tt.want {
				t.Errorf("tailLines(%q, %d) = %q, want %q", tt.input, tt.n, got, tt.want)
			}
		})
	}
}

func TestDeployHandler_Auth(t *testing.T) {
	h := &deployHandler{
		secret:      "test-secret",
		runPipeline: func() (bool, string) { return true, "" },
	}

	t.Run("wrong secret returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/deploy", nil)
		req.Header.Set("Authorization", "Bearer wrong")
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("got %d, want 401", rr.Code)
		}
	})

	t.Run("missing auth header returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/deploy", nil)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("got %d, want 401", rr.Code)
		}
	})

	t.Run("correct secret returns 200 with ok body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/deploy", nil)
		req.Header.Set("Authorization", "Bearer test-secret")
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("got %d, want 200", rr.Code)
		}
		var result deployResult
		if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
			t.Fatalf("could not decode response: %v", err)
		}
		if !result.OK {
			t.Errorf("expected ok=true, got ok=false")
		}
	})
}

func TestDeployHandler_InProgress(t *testing.T) {
	started := make(chan struct{})
	done := make(chan struct{})
	t.Cleanup(func() { close(done) })
	h := &deployHandler{
		secret: "s",
		runPipeline: func() (bool, string) {
			close(started)
			<-done
			return true, ""
		},
	}

	go func() {
		req := httptest.NewRequest(http.MethodPost, "/deploy", nil)
		req.Header.Set("Authorization", "Bearer s")
		h.ServeHTTP(httptest.NewRecorder(), req)
	}()

	<-started

	req := httptest.NewRequest(http.MethodPost, "/deploy", nil)
	req.Header.Set("Authorization", "Bearer s")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusConflict {
		t.Errorf("in-progress: got %d, want 409", rr.Code)
	}
}
