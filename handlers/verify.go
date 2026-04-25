package handlers

import (
	"encoding/json"
	"net/http"
)

// BuildSHA256, BuildVersion, and BuildDate are set via ldflags at build time.
// e.g. -ldflags "-X handlers.BuildSHA256=abc123 -X handlers.BuildVersion=v1.0.0"
var (
	BuildSHA256  = "unknown"
	BuildVersion = "dev"
	BuildDate    = "unknown"
)

// Verify handles GET /api/v1/verify
func Verify(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{
		"sha256":  BuildSHA256,
		"version": BuildVersion,
		"built":   BuildDate,
	})
}
