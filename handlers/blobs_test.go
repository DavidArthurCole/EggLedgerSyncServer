package handlers_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/DavidArthurCole/EggLedgerSyncServer/handlers"
)

func newMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db, mock
}

func blobsReq(method, path, discordID, name string, body []byte) *http.Request {
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, bytes.NewReader(body))
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	if discordID != "" {
		req.Header.Set("X-Discord-ID", discordID)
	}
	if name != "" {
		req.SetPathValue("name", name)
	}
	return req
}

// Put

func TestBlobPut_Success(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectExec("INSERT INTO users").
		WithArgs("u1", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("INSERT INTO blobs").
		WithArgs("u1", "accounts", "cipher==", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	body, _ := json.Marshal(map[string]string{"ciphertext": "cipher=="})
	rr := httptest.NewRecorder()
	handlers.NewBlobHandlers(db).Put(rr, blobsReq(http.MethodPut, "/", "u1", "accounts", body))

	if rr.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rr.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestBlobPut_BadJSON(t *testing.T) {
	db, _ := newMockDB(t)
	rr := httptest.NewRecorder()
	handlers.NewBlobHandlers(db).Put(rr, blobsReq(http.MethodPut, "/", "u1", "accounts", []byte("not-json")))

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestBlobPut_DBError(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectExec("INSERT INTO users").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("INSERT INTO blobs").
		WillReturnError(sql.ErrConnDone)

	body, _ := json.Marshal(map[string]string{"ciphertext": "c"})
	rr := httptest.NewRecorder()
	handlers.NewBlobHandlers(db).Put(rr, blobsReq(http.MethodPut, "/", "u1", "accounts", body))

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
	if rr.Body.String() == sql.ErrConnDone.Error()+"\n" {
		t.Error("raw DB error must not be sent to client")
	}
}

// Get

func TestBlobGet_Success(t *testing.T) {
	db, mock := newMockDB(t)
	rows := sqlmock.NewRows([]string{"ciphertext", "updated_at"}).AddRow("cipher==", int64(1000))
	mock.ExpectQuery("SELECT ciphertext").
		WithArgs("u1", "accounts").
		WillReturnRows(rows)

	rr := httptest.NewRecorder()
	handlers.NewBlobHandlers(db).Get(rr, blobsReq(http.MethodGet, "/", "u1", "accounts", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var body map[string]any
	json.NewDecoder(rr.Body).Decode(&body)
	if body["ciphertext"] != "cipher==" {
		t.Errorf("ciphertext: got %v", body["ciphertext"])
	}
	if rr.Header().Get("Content-Type") != "application/json" {
		t.Error("missing Content-Type: application/json")
	}
}

func TestBlobGet_NotFound(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectQuery("SELECT ciphertext").
		WithArgs("u1", "missing").
		WillReturnError(sql.ErrNoRows)

	rr := httptest.NewRecorder()
	handlers.NewBlobHandlers(db).Get(rr, blobsReq(http.MethodGet, "/", "u1", "missing", nil))

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestBlobGet_DBError(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectQuery("SELECT ciphertext").
		WillReturnError(sql.ErrConnDone)

	rr := httptest.NewRecorder()
	handlers.NewBlobHandlers(db).Get(rr, blobsReq(http.MethodGet, "/", "u1", "accounts", nil))

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}

// List

func TestBlobList_Empty(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectQuery("SELECT name").
		WithArgs("u1").
		WillReturnRows(sqlmock.NewRows([]string{"name", "updated_at"}))

	rr := httptest.NewRecorder()
	handlers.NewBlobHandlers(db).List(rr, blobsReq(http.MethodGet, "/", "u1", "", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	// Must be [] not null
	if body := rr.Body.String(); body != "[]\n" {
		t.Errorf("empty list must encode as [], got %q", body)
	}
}

func TestBlobList_Multiple(t *testing.T) {
	db, mock := newMockDB(t)
	rows := sqlmock.NewRows([]string{"name", "updated_at"}).
		AddRow("accounts", int64(100)).
		AddRow("reports", int64(200))
	mock.ExpectQuery("SELECT name").
		WithArgs("u1").
		WillReturnRows(rows)

	rr := httptest.NewRecorder()
	handlers.NewBlobHandlers(db).List(rr, blobsReq(http.MethodGet, "/", "u1", "", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var items []map[string]any
	json.NewDecoder(rr.Body).Decode(&items)
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
	if items[0]["name"] != "accounts" || items[1]["name"] != "reports" {
		t.Errorf("unexpected items: %v", items)
	}
}

func TestBlobList_DBError(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectQuery("SELECT name").
		WillReturnError(sql.ErrConnDone)

	rr := httptest.NewRecorder()
	handlers.NewBlobHandlers(db).List(rr, blobsReq(http.MethodGet, "/", "u1", "", nil))

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}

// Delete

func TestBlobDelete_Success(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectExec("DELETE FROM blobs").
		WithArgs("u1", "accounts").
		WillReturnResult(sqlmock.NewResult(0, 1))

	rr := httptest.NewRecorder()
	handlers.NewBlobHandlers(db).Delete(rr, blobsReq(http.MethodDelete, "/", "u1", "accounts", nil))

	if rr.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rr.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestBlobDelete_NotExist(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectExec("DELETE FROM blobs").
		WithArgs("u1", "ghost").
		WillReturnResult(sqlmock.NewResult(0, 0))

	rr := httptest.NewRecorder()
	handlers.NewBlobHandlers(db).Delete(rr, blobsReq(http.MethodDelete, "/", "u1", "ghost", nil))

	// Idempotent - still 204 even if nothing was deleted
	if rr.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rr.Code)
	}
}

// DeleteUser

func TestBlobDeleteUser_Success(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectExec("DELETE FROM users").
		WithArgs("u1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	rr := httptest.NewRecorder()
	handlers.NewBlobHandlers(db).DeleteUser(rr, blobsReq(http.MethodDelete, "/", "u1", "", nil))

	if rr.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rr.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}
