package notes

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// ANCHOR: test

func TestGetNote(t *testing.T) {
	s := NewServer("s3cret", map[string]string{"1": "buy milk"})
	req := httptest.NewRequest(http.MethodGet, "/notes/1", nil)
	req.SetPathValue("id", "1")
	rec := httptest.NewRecorder()

	s.getNote(rec, req)

	if rec.Code != http.StatusOK || rec.Body.String() != "buy milk" {
		t.Errorf("getNote = %d %q, want 200 %q", rec.Code, rec.Body.String(), "buy milk")
	}
}

func TestDeleteNote(t *testing.T) {
	s := NewServer("s3cret", map[string]string{"1": "buy milk"})
	req := httptest.NewRequest(http.MethodDelete, "/notes/1", nil)
	req.SetPathValue("id", "1")
	rec := httptest.NewRecorder()

	s.deleteNote(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("deleteNote = %d, want %d", rec.Code, http.StatusNoContent)
	}
}

func TestRequireToken(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	h := requireToken("s3cret", next)
	req := httptest.NewRequest(http.MethodDelete, "/notes/1", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("requireToken without token = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

// ANCHOR_END: test
