package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

type BlobHandlers struct{ db *sql.DB }

func NewBlobHandlers(db *sql.DB) *BlobHandlers { return &BlobHandlers{db: db} }

func (h *BlobHandlers) Put(w http.ResponseWriter, r *http.Request) {
	discordID := r.Header.Get("X-Discord-ID")
	name := r.PathValue("name")

	var body struct {
		Ciphertext string `json:"ciphertext"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	h.db.ExecContext(r.Context(),
		`INSERT INTO users (discord_id, created_at) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		discordID, time.Now().Unix(),
	)

	_, err := h.db.ExecContext(r.Context(),
		`INSERT INTO blobs (discord_id, name, ciphertext, updated_at) VALUES ($1, $2, $3, $4)
         ON CONFLICT (discord_id, name) DO UPDATE SET ciphertext = EXCLUDED.ciphertext, updated_at = EXCLUDED.updated_at`,
		discordID, name, body.Ciphertext, time.Now().Unix(),
	)
	if err != nil {
		log.Printf("blobs.Put: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *BlobHandlers) Get(w http.ResponseWriter, r *http.Request) {
	discordID := r.Header.Get("X-Discord-ID")
	name := r.PathValue("name")

	var ciphertext string
	var updatedAt int64
	err := h.db.QueryRowContext(r.Context(),
		`SELECT ciphertext, updated_at FROM blobs WHERE discord_id = $1 AND name = $2`,
		discordID, name,
	).Scan(&ciphertext, &updatedAt)
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("blobs.Get: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"ciphertext": ciphertext,
		"updatedAt":  updatedAt,
	})
}

func (h *BlobHandlers) List(w http.ResponseWriter, r *http.Request) {
	discordID := r.Header.Get("X-Discord-ID")
	rows, err := h.db.QueryContext(r.Context(),
		`SELECT name, updated_at FROM blobs WHERE discord_id = $1`, discordID,
	)
	if err != nil {
		log.Printf("blobs.List: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type entry struct {
		Name      string `json:"name"`
		UpdatedAt int64  `json:"updatedAt"`
	}
	items := []entry{}
	for rows.Next() {
		var e entry
		if err := rows.Scan(&e.Name, &e.UpdatedAt); err != nil {
			log.Printf("blobs.List scan: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		items = append(items, e)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

func (h *BlobHandlers) Delete(w http.ResponseWriter, r *http.Request) {
	discordID := r.Header.Get("X-Discord-ID")
	name := r.PathValue("name")
	h.db.ExecContext(r.Context(),
		`DELETE FROM blobs WHERE discord_id = $1 AND name = $2`, discordID, name,
	)
	w.WriteHeader(http.StatusNoContent)
}

func (h *BlobHandlers) DeleteUser(w http.ResponseWriter, r *http.Request) {
	discordID := r.Header.Get("X-Discord-ID")
	h.db.ExecContext(r.Context(), `DELETE FROM users WHERE discord_id = $1`, discordID)
	w.WriteHeader(http.StatusNoContent)
}
