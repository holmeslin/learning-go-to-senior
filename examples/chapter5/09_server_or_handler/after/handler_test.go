package notes_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	notes "github.com/holmeslin/learning-go-to-senior/examples/chapter5/09_server_or_handler/after"
)

const token = "s3cret"

// newHandler 每次呼叫都建一個全新的 Handler，測試之間不共用狀態。
func newHandler() http.Handler {
	return notes.NewHandler(token, map[string]string{"1": "buy milk"})
}

// ANCHOR: table

func TestRoutes(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		auth   string
		want   int
	}{
		{"get existing note", "GET", "/notes/1", "", http.StatusOK},
		{"get missing note", "GET", "/notes/9", "", http.StatusNotFound},
		{"unknown path", "GET", "/note/1", "", http.StatusNotFound},
		{"wrong method", "POST", "/notes/1", "", http.StatusMethodNotAllowed},
		{"delete without token", "DELETE", "/notes/1", "", http.StatusUnauthorized},
		{"delete with wrong token", "DELETE", "/notes/1", "Bearer nope", http.StatusUnauthorized},
		{"delete with token", "DELETE", "/notes/1", "Bearer " + token, http.StatusNoContent},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := do(newHandler(), tt.method, tt.path, tt.auth); got != tt.want {
				t.Errorf("%s %s = %d, want %d", tt.method, tt.path, got, tt.want)
			}
		})
	}
}

// ANCHOR_END: table

func TestGetNoteBody(t *testing.T) {
	rec := httptest.NewRecorder()
	newHandler().ServeHTTP(rec, httptest.NewRequest("GET", "/notes/1", nil))

	if got, want := rec.Body.String(), "buy milk"; got != want {
		t.Errorf("GET /notes/1 body = %q, want %q", got, want)
	}
}

func TestUnauthorizedDeleteKeepsNote(t *testing.T) {
	h := newHandler()
	do(h, "DELETE", "/notes/1", "")

	if got := do(h, "GET", "/notes/1", ""); got != http.StatusOK {
		t.Errorf("GET /notes/1 after unauthorized delete = %d, want %d", got, http.StatusOK)
	}
}

func TestDeleteRemovesNote(t *testing.T) {
	h := newHandler()
	do(h, "DELETE", "/notes/1", "Bearer "+token)

	if got := do(h, "GET", "/notes/1", ""); got != http.StatusNotFound {
		t.Errorf("GET /notes/1 after delete = %d, want %d", got, http.StatusNotFound)
	}
}

// ANCHOR: client

func TestClientGet(t *testing.T) {
	srv := httptest.NewServer(newHandler())
	defer srv.Close()
	c := &notes.Client{BaseURL: srv.URL, HTTP: srv.Client()}

	got, err := c.Get(t.Context(), "1")
	if err != nil || got != "buy milk" {
		t.Errorf("Get(1) = %q, %v; want %q, nil", got, err, "buy milk")
	}
	if _, err := c.Get(t.Context(), "9"); !errors.Is(err, notes.ErrNotFound) {
		t.Errorf("Get(9) error = %v, want %v", err, notes.ErrNotFound)
	}
}

func TestClientGetCanceled(t *testing.T) {
	started := make(chan struct{}, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started <- struct{}{}
		<-r.Context().Done() // 卡到 client 斷線為止
	}))
	defer srv.Close()
	c := &notes.Client{BaseURL: srv.URL, HTTP: srv.Client()}

	ctx, cancel := context.WithCancel(t.Context())
	errc := make(chan error, 1)
	go func() {
		_, err := c.Get(ctx, "1")
		errc <- err
	}()
	<-started // server 已經收到請求，再取消
	cancel()

	if err := <-errc; !errors.Is(err, context.Canceled) {
		t.Errorf("Get after cancel: error = %v, want %v", err, context.Canceled)
	}
}

// ANCHOR_END: client
