package health_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	health "github.com/holmeslin/learning-go-to-senior/examples/chapter9/07_health_check/after"
)

// ANCHOR: checks

// get 對組好的 handler 發一個 GET，回傳狀態碼與去掉換行的 body。
func get(h http.Handler, path string) (int, string) {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec.Code, strings.TrimSpace(rec.Body.String())
}

// hangs 模擬卡住的下游：只在 ctx 結束時才回來。
func hangs(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}

// refuses 模擬直接失敗的下游。
func refuses(context.Context) error { return errors.New("connection refused") }

// ANCHOR_END: checks

// ANCHOR: readyz

func TestReadyzListsFailedChecks(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := health.NewHandler(
			health.Check{Name: "db", Timeout: 2 * time.Second, Probe: hangs},
			health.Check{Name: "cache", Timeout: 3 * time.Second, Probe: refuses},
			health.Check{Name: "queue", Timeout: time.Second, Probe: hangs},
		)

		start := time.Now()
		code, body := get(h, "/readyz")

		if code != http.StatusServiceUnavailable {
			t.Errorf("GET /readyz = %d, want 503", code)
		}
		want := `{"status":"unavailable","failed":{"cache":"connection refused","db":"timeout","queue":"timeout"}}`
		if body != want {
			t.Errorf("body = %s, want %s", body, want)
		}
		if got := time.Since(start); got != 2*time.Second {
			t.Errorf("GET /readyz took %v, want 2s (the slowest check; run one after another it would be 3s)", got)
		}
	})
}

// ANCHOR_END: readyz

// ANCHOR: stuck

func TestReadyzDoesNotWaitForProbeThatIgnoresContext(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		release := make(chan struct{})
		defer close(release) // bubble 結束前，卡住的 Probe 必須能離開
		h := health.NewHandler(health.Check{
			Name: "db", Timeout: time.Second,
			Probe: func(context.Context) error { <-release; return nil },
		})

		code, body := get(h, "/readyz")

		if code != http.StatusServiceUnavailable || !strings.Contains(body, `"db":"timeout"`) {
			t.Errorf("GET /readyz = %d %s, want 503 with db timeout", code, body)
		}
	})
}

// ANCHOR_END: stuck

// ANCHOR: livez

func TestLivezIgnoresDownstream(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := health.NewHandler(
			health.Check{Name: "db", Timeout: time.Minute, Probe: hangs},
			health.Check{Name: "cache", Timeout: time.Minute, Probe: refuses},
		)

		start := time.Now()
		code, body := get(h, "/livez")

		if code != http.StatusOK || body != "ok" {
			t.Errorf("GET /livez = %d %q, want 200 \"ok\"", code, body)
		}
		if got := time.Since(start); got != 0 {
			t.Errorf("GET /livez took %v in the bubble, want 0 (no downstream call)", got)
		}
		if code, _ := get(h, "/readyz"); code != http.StatusServiceUnavailable {
			t.Errorf("GET /readyz = %d, want 503 while /livez is 200", code)
		}
	})
}

// ANCHOR_END: livez

func TestReadyzHealthy(t *testing.T) {
	t.Parallel()
	ok := func(context.Context) error { return nil }
	h := health.NewHandler(health.Check{Name: "db", Timeout: time.Second, Probe: ok})

	if code, body := get(h, "/readyz"); code != http.StatusOK || body != `{"status":"ok"}` {
		t.Errorf("GET /readyz = %d %s, want 200 {\"status\":\"ok\"}", code, body)
	}
	if code, _ := get(health.NewHandler(), "/readyz"); code != http.StatusOK {
		t.Errorf("GET /readyz without checks = %d, want 200", code)
	}
}
