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
	_flagAddr        = flag.String("addr", ":8080", "listen address")
	_flagDBConnStr   = flag.String("db", os.Getenv("DATABASE_URL"), "PostgreSQL connection string (postgres://user:pass@host:5432/dbname)")
	_flagDiscordID   = flag.String("discord-client-id", os.Getenv("DISCORD_CLIENT_ID"), "Discord OAuth2 client ID")
	_flagDiscordSec  = flag.String("discord-client-secret", os.Getenv("DISCORD_CLIENT_SECRET"), "Discord OAuth2 client secret")
	_flagRedirectURL = flag.String("redirect-url", "https://ledgersync.davidarthurcole.me/api/v1/auth/callback", "OAuth2 redirect URL")
)

func main() {
	flag.Parse()

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

	err := auth.HandleCallback(r.Context(), code, state, func(state, token, discordID string) error {
		database.ExecContext(r.Context(),
			`INSERT INTO users (discord_id, created_at) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			discordID, time.Now().Unix(),
		)
		_, err := database.ExecContext(r.Context(),
			`INSERT INTO sessions (token, discord_id, expires_at) VALUES ($1, $2, $3)`,
			token, discordID, time.Now().Add(24*time.Hour).Unix(),
		)
		if err != nil {
			return err
		}
		_, err = database.ExecContext(r.Context(),
			`UPDATE pending_auth SET session_token = $1 WHERE state = $2`, token, state,
		)
		return err
	})
	if err != nil {
		http.Error(w, "auth failed", http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "<html><body>Authentication successful! You may close this window.</body></html>")
}

func handleAuthPoll(w http.ResponseWriter, r *http.Request, database *sql.DB) {
	state := r.URL.Query().Get("state")
	var token sql.NullString
	var expiresAt int64
	err := database.QueryRowContext(r.Context(),
		`SELECT session_token, expires_at FROM pending_auth WHERE state = $1`, state,
	).Scan(&token, &expiresAt)
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
	json.NewEncoder(w).Encode(map[string]string{"token": token.String})
}

func handleDeleteSession(w http.ResponseWriter, r *http.Request, database *sql.DB) {
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if token != "" {
		database.ExecContext(r.Context(), `DELETE FROM sessions WHERE token = $1`, token)
	}
	w.WriteHeader(http.StatusNoContent)
}
