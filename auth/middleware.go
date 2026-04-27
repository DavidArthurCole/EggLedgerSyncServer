package auth

import (
	"database/sql"
	"net/http"
	"strings"
	"time"
)

// RequireAuth is an HTTP middleware that validates the Bearer token from the DB.
func RequireAuth(db *sql.DB, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		token := strings.TrimPrefix(header, "Bearer ")
		if token == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		var discordID string
		var expiresAt int64
		err := db.QueryRowContext(r.Context(),
			`SELECT discord_id, expires_at FROM sessions WHERE token = $1`, token,
		).Scan(&discordID, &expiresAt)
		if err != nil || time.Now().Unix() > expiresAt {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		// Sliding expiry: every successful authenticated request resets the clock to 30 days from now.
		db.ExecContext(r.Context(),
			`UPDATE sessions SET expires_at = $1 WHERE token = $2`,
			time.Now().Add(30*24*time.Hour).Unix(), token,
		)
		r.Header.Set("X-Discord-ID", discordID)
		next(w, r)
	}
}
