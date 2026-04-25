package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DavidArthurCole/EggLedgerSyncServer/handlers"
)

func TestVerify_ReturnsFields(t *testing.T) {
	handlers.BuildSHA256 = "abc123"
	handlers.BuildVersion = "v1.2.3"
	handlers.BuildDate = "2025-01-01T00:00:00Z"

	req := httptest.NewRequest(http.MethodGet, "/api/v1/verify", nil)
	rr := httptest.NewRecorder()
	handlers.Verify(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["sha256"] != "abc123" {
		t.Errorf("sha256: got %q, want %q", body["sha256"], "abc123")
	}
	if body["version"] != "v1.2.3" {
		t.Errorf("version: got %q, want %q", body["version"], "v1.2.3")
	}
	if body["built"] != "2025-01-01T00:00:00Z" {
		t.Errorf("built: got %q, want %q", body["built"], "2025-01-01T00:00:00Z")
	}
}

func TestVerify_DefaultsToUnknown(t *testing.T) {
	handlers.BuildSHA256 = "unknown"
	handlers.BuildVersion = "dev"
	handlers.BuildDate = "unknown"

	req := httptest.NewRequest(http.MethodGet, "/api/v1/verify", nil)
	rr := httptest.NewRecorder()
	handlers.Verify(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var body map[string]string
	json.NewDecoder(rr.Body).Decode(&body)
	if body["sha256"] != "unknown" || body["version"] != "dev" {
		t.Errorf("unexpected defaults: %v", body)
	}
}
