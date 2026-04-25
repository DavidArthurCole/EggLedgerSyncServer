package auth_test

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/DavidArthurCole/EggLedgerSyncServer/auth"
)

func TestRequireAuth_NoHeader(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	reached := false
	handler := auth.RequireAuth(db, func(w http.ResponseWriter, r *http.Request) {
		reached = true
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
	if reached {
		t.Error("inner handler must not be called without auth header")
	}
}

func TestRequireAuth_EmptyBearerToken(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	handler := auth.RequireAuth(db, func(w http.ResponseWriter, r *http.Request) {})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer ")
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for empty bearer token, got %d", rr.Code)
	}
}

func TestRequireAuth_ValidToken(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	future := time.Now().Add(time.Hour).Unix()
	rows := sqlmock.NewRows([]string{"discord_id", "expires_at"}).AddRow("user123", future)
	mock.ExpectQuery("SELECT discord_id").
		WithArgs("valid-token").
		WillReturnRows(rows)

	var gotDiscordID string
	handler := auth.RequireAuth(db, func(w http.ResponseWriter, r *http.Request) {
		gotDiscordID = r.Header.Get("X-Discord-ID")
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if gotDiscordID != "user123" {
		t.Errorf("X-Discord-ID: got %q, want %q", gotDiscordID, "user123")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestRequireAuth_ExpiredToken(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	past := time.Now().Add(-time.Hour).Unix()
	rows := sqlmock.NewRows([]string{"discord_id", "expires_at"}).AddRow("user123", past)
	mock.ExpectQuery("SELECT discord_id").
		WithArgs("old-token").
		WillReturnRows(rows)

	reached := false
	handler := auth.RequireAuth(db, func(w http.ResponseWriter, r *http.Request) {
		reached = true
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer old-token")
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for expired token, got %d", rr.Code)
	}
	if reached {
		t.Error("inner handler must not be called with expired token")
	}
}

func TestRequireAuth_UnknownToken(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT discord_id").
		WithArgs("unknown-token").
		WillReturnError(sql.ErrNoRows)

	reached := false
	handler := auth.RequireAuth(db, func(w http.ResponseWriter, r *http.Request) {
		reached = true
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer unknown-token")
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for unknown token, got %d", rr.Code)
	}
	if reached {
		t.Error("inner handler must not be called with unknown token")
	}
}
