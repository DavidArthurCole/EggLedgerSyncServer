package auth

import (
	"encoding/hex"
	"testing"
)

func TestRandomHex_Length(t *testing.T) {
	for _, n := range []int{8, 16, 32} {
		got := randomHex(n)
		// n bytes -> 2n hex chars
		if len(got) != n*2 {
			t.Errorf("randomHex(%d): got length %d, want %d", n, len(got), n*2)
		}
		if _, err := hex.DecodeString(got); err != nil {
			t.Errorf("randomHex(%d) = %q: not valid hex: %v", n, got, err)
		}
	}
}

func TestRandomHex_Unique(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		v := randomHex(16)
		if seen[v] {
			t.Fatalf("randomHex(16) collision after %d calls: %q", i, v)
		}
		seen[v] = true
	}
}

func TestAuthURL_ReturnsStateAndURL(t *testing.T) {
	Init("test-client-id", "test-client-secret", "https://example.com/callback")

	url, state := AuthURL()
	if url == "" {
		t.Error("AuthURL returned empty url")
	}
	if state == "" {
		t.Error("AuthURL returned empty state")
	}
	if len(state) != 32 {
		t.Errorf("state length: got %d, want 32", len(state))
	}
}

func TestAuthURL_StateUniquePerCall(t *testing.T) {
	Init("test-client-id", "test-client-secret", "https://example.com/callback")

	_, s1 := AuthURL()
	_, s2 := AuthURL()
	if s1 == s2 {
		t.Error("two consecutive AuthURL calls returned the same state")
	}
}
