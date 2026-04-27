package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
	_ "time/tzdata"

	"github.com/DavidArthurCole/EggLedgerSyncServer/auth"
	"github.com/DavidArthurCole/EggLedgerSyncServer/db"
	"github.com/DavidArthurCole/EggLedgerSyncServer/handlers"
)

var (
	_flagAddr        = flag.String("addr", os.Getenv("LISTEN_ADDR"), "listen address (env: LISTEN_ADDR)")
	_flagDBConnStr   = flag.String("db", os.Getenv("DATABASE_URL"), "PostgreSQL connection string (postgres://user:pass@host:5432/dbname)")
	_flagDiscordID   = flag.String("discord-client-id", os.Getenv("DISCORD_CLIENT_ID"), "Discord OAuth2 client ID")
	_flagDiscordSec  = flag.String("discord-client-secret", os.Getenv("DISCORD_CLIENT_SECRET"), "Discord OAuth2 client secret")
	_flagRedirectURL = flag.String("redirect-url", "https://ledgersync.davidarthurcole.me/api/v1/auth/callback", "OAuth2 redirect URL")
)

func main() {
	flag.Parse()

	if *_flagAddr == "" {
		*_flagAddr = ":8080"
	}

	if err := db.Init(*_flagDBConnStr); err != nil {
		log.Fatalf("db init: %v", err)
	}
	defer db.Close()

	auth.Init(*_flagDiscordID, *_flagDiscordSec, *_flagRedirectURL)

	mux := http.NewServeMux()
	blobs := handlers.NewBlobHandlers(db.DB())

	mux.HandleFunc("GET /api/v1/auth/discord", handleAuthDiscord)
	mux.HandleFunc("GET /api/v1/auth/callback", func(w http.ResponseWriter, r *http.Request) {
		handleAuthCallback(w, r, db.DB())
	})
	mux.HandleFunc("GET /api/v1/auth/poll", func(w http.ResponseWriter, r *http.Request) {
		handleAuthPoll(w, r, db.DB())
	})
	mux.HandleFunc("DELETE /api/v1/auth/session", func(w http.ResponseWriter, r *http.Request) {
		handleDeleteSession(w, r, db.DB())
	})

	mux.HandleFunc("PUT /api/v1/blobs/{name}", auth.RequireAuth(db.DB(), blobs.Put))
	mux.HandleFunc("GET /api/v1/blobs/{name}", auth.RequireAuth(db.DB(), blobs.Get))
	mux.HandleFunc("GET /api/v1/blobs", auth.RequireAuth(db.DB(), blobs.List))
	mux.HandleFunc("DELETE /api/v1/blobs/{name}", auth.RequireAuth(db.DB(), blobs.Delete))
	mux.HandleFunc("DELETE /api/v1/user", auth.RequireAuth(db.DB(), blobs.DeleteUser))

	mux.HandleFunc("GET /api/v1/verify", handlers.Verify)

	srv := &http.Server{
		Addr:         *_flagAddr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go func() {
		log.Printf("listening on %s", *_flagAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}

func handleAuthDiscord(w http.ResponseWriter, r *http.Request) {
	url, state := auth.AuthURL()
	db.DB().ExecContext(r.Context(),
		`INSERT INTO pending_auth (state, expires_at) VALUES ($1, $2) ON CONFLICT (state) DO UPDATE SET expires_at = EXCLUDED.expires_at`,
		state, time.Now().Add(10*time.Minute).Unix(),
	)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"url": url, "state": state})
}

func handleAuthCallback(w http.ResponseWriter, r *http.Request, database *sql.DB) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	err := auth.HandleCallback(r.Context(), code, state, func(state, token string, user auth.DiscordUser) error {
		database.ExecContext(r.Context(),
			`INSERT INTO users (discord_id, created_at, username, avatar_url) VALUES ($1, $2, $3, $4)
			 ON CONFLICT (discord_id) DO UPDATE SET username = EXCLUDED.username, avatar_url = EXCLUDED.avatar_url`,
			user.ID, time.Now().Unix(), user.Username, user.AvatarURL,
		)
		// Ensure the user has a stable encryption key. Generate one on first auth only.
		var encKey string
		database.QueryRowContext(r.Context(),
			`SELECT encryption_key FROM users WHERE discord_id = $1`, user.ID,
		).Scan(&encKey)
		if encKey == "" {
			encKey = auth.GenerateEncryptionKey()
			database.ExecContext(r.Context(),
				`UPDATE users SET encryption_key = $1 WHERE discord_id = $2`, encKey, user.ID,
			)
		}
		_, err := database.ExecContext(r.Context(),
			`INSERT INTO sessions (token, discord_id, expires_at) VALUES ($1, $2, $3)`,
			token, user.ID, time.Now().Add(30*24*time.Hour).Unix(),
		)
		if err != nil {
			return err
		}
		_, err = database.ExecContext(r.Context(),
			`UPDATE pending_auth SET session_token = $1, username = $2, avatar_url = $3, encryption_key = $4 WHERE state = $5`,
			token, user.Username, user.AvatarURL, encKey, state,
		)
		return err
	})
	if err != nil {
		http.Error(w, "auth failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Authentication Successful</title>
<style>
  *{margin:0;padding:0;box-sizing:border-box}
  body{min-height:100vh;display:flex;align-items:center;justify-content:center;
    background:#1a1a2e;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif}
  .card{background:#16213e;border:1px solid #0f3460;border-radius:12px;
    padding:48px 40px;text-align:center;max-width:380px;width:90%}
  .check{width:64px;height:64px;background:#22c55e;border-radius:50%;
    display:flex;align-items:center;justify-content:center;
    margin:0 auto 24px;font-size:28px;color:#fff}
  h1{color:#f1f5f9;font-size:22px;font-weight:600;margin-bottom:12px}
  p{color:#94a3b8;font-size:14px;line-height:1.6}
</style>
</head>
<body>
<div class="card">
  <div class="check">&#10003;</div>
  <h1>Authentication Successful</h1>
  <p>You're all set. You can close this window and return to EggLedger.</p>
</div>
</body>
</html>`)
}

func handleAuthPoll(w http.ResponseWriter, r *http.Request, database *sql.DB) {
	state := r.URL.Query().Get("state")
	var token sql.NullString
	var username, avatarURL, encryptionKey string
	var expiresAt int64
	err := database.QueryRowContext(r.Context(),
		`SELECT session_token, expires_at, username, avatar_url, encryption_key FROM pending_auth WHERE state = $1`, state,
	).Scan(&token, &expiresAt, &username, &avatarURL, &encryptionKey)
	if err != nil || time.Now().Unix() > expiresAt {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if !token.Valid {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	database.ExecContext(r.Context(), `DELETE FROM pending_auth WHERE state = $1`, state)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": token.String, "username": username, "avatarUrl": avatarURL, "encryptionKey": encryptionKey})
}

func handleDeleteSession(w http.ResponseWriter, r *http.Request, database *sql.DB) {
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if token != "" {
		database.ExecContext(r.Context(), `DELETE FROM sessions WHERE token = $1`, token)
	}
	w.WriteHeader(http.StatusNoContent)
}
