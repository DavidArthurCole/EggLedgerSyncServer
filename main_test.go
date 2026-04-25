package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

// handleDeleteSession

func TestHandleDeleteSession_EmptyHeader(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/auth/session", nil)
	rr := httptest.NewRecorder()
	handleDeleteSession(rr, req, db)

	if rr.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rr.Code)
	}
}

// Regression: header[len("Bearer "):] panicked on short strings.
func TestHandleDeleteSession_ShortHeader(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	for _, hdr := range []string{"x", "Bear", "Bearer"} {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/auth/session", nil)
		req.Header.Set("Authorization", hdr)
		rr := httptest.NewRecorder()

		// Must not panic
		handleDeleteSession(rr, req, db)

		if rr.Code != http.StatusNoContent {
			t.Errorf("header=%q: expected 204, got %d", hdr, rr.Code)
		}
	}
}

func TestHandleDeleteSession_ValidToken(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectExec("DELETE FROM sessions").
		WithArgs("my-token").
		WillReturnResult(sqlmock.NewResult(0, 1))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/auth/session", nil)
	req.Header.Set("Authorization", "Bearer my-token")
	rr := httptest.NewRecorder()
	handleDeleteSession(rr, req, db)

	if rr.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rr.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

// handleAuthPoll

func TestHandleAuthPoll_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT session_token").
		WithArgs("bad-state").
		WillReturnError(sql.ErrNoRows)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/poll?state=bad-state", nil)
	rr := httptest.NewRecorder()
	handleAuthPoll(rr, req, db)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestHandleAuthPoll_Expired(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	past := time.Now().Add(-time.Hour).Unix()
	rows := sqlmock.NewRows([]string{"session_token", "expires_at"}).
		AddRow(nil, past)
	mock.ExpectQuery("SELECT session_token").
		WithArgs("expired-state").
		WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/poll?state=expired-state", nil)
	rr := httptest.NewRecorder()
	handleAuthPoll(rr, req, db)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 for expired state, got %d", rr.Code)
	}
}

func TestHandleAuthPoll_Pending(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	future := time.Now().Add(time.Hour).Unix()
	// session_token is NULL - auth not yet complete
	rows := sqlmock.NewRows([]string{"session_token", "expires_at"}).
		AddRow(nil, future)
	mock.ExpectQuery("SELECT session_token").
		WithArgs("pending-state").
		WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/poll?state=pending-state", nil)
	rr := httptest.NewRecorder()
	handleAuthPoll(rr, req, db)

	if rr.Code != http.StatusAccepted {
		t.Errorf("expected 202 while pending, got %d", rr.Code)
	}
}

func TestHandleAuthPoll_Ready(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	future := time.Now().Add(time.Hour).Unix()
	rows := sqlmock.NewRows([]string{"session_token", "expires_at"}).
		AddRow("session-abc", future)
	mock.ExpectQuery("SELECT session_token").
		WithArgs("ready-state").
		WillReturnRows(rows)
	mock.ExpectExec("DELETE FROM pending_auth").
		WithArgs("ready-state").
		WillReturnResult(sqlmock.NewResult(0, 1))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/poll?state=ready-state", nil)
	rr := httptest.NewRecorder()
	handleAuthPoll(rr, req, db)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var body map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["token"] != "session-abc" {
		t.Errorf("token: got %q, want %q", body["token"], "session-abc")
	}
	if rr.Header().Get("Content-Type") != "application/json" {
		t.Error("missing Content-Type: application/json")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}
