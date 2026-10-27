package taskapi_test

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	taskapi "github.com/holmeslin/learning-go-to-senior/examples/chapter2/03_handle_at_boundary/after"
)

func TestHandlerLogsFailureOnce(t *testing.T) {
	var buf bytes.Buffer
	svc := &taskapi.Service{Store: &taskapi.Store{Down: true}}
	h := taskapi.NewHandler(svc, log.New(&buf, "", 0))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("POST", "/tasks/7/complete", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
	if n := strings.Count(buf.String(), "\n"); n != 1 {
		t.Errorf("log lines = %d, want 1:\n%s", n, buf.String())
	}
}

func TestHandlerSuccessLogsNothing(t *testing.T) {
	var buf bytes.Buffer
	svc := &taskapi.Service{Store: &taskapi.Store{}}
	h := taskapi.NewHandler(svc, log.New(&buf, "", 0))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("POST", "/tasks/7/complete", nil))

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", rec.Code)
	}
	if buf.Len() != 0 {
		t.Errorf("unexpected log output: %q", buf.String())
	}
}

func TestHandlerBadID(t *testing.T) {
	svc := &taskapi.Service{Store: &taskapi.Store{}}
	h := taskapi.NewHandler(svc, log.New(&bytes.Buffer{}, "", 0))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("POST", "/tasks/abc/complete", nil))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}
