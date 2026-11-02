package slowprovider_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter16/reminder/internal/slowprovider"
)

// serve 呼叫 h 一次。回傳的 recorder 的 Code 從 0 開始（httptest.NewRecorder 會預設成 200），
// 所以 Code 仍是 0 代表 handler 沒有回應。
func serve(ctx context.Context, h http.Handler, method string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	rec.Code = 0
	req := httptest.NewRequestWithContext(ctx, method, "/push", strings.NewReader(`{}`))
	h.ServeHTTP(rec, req)
	return rec
}

// 每個請求先等 Delay 再回應；回 429 還是 200 由亂數與比例決定。
// 時間在 synctest 的 bubble 裡是假的，所以延遲是確定的、測試不用真的等。
func TestHandlerWaitsThenAnswersByTheShare(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		rolls := []float64{0.05, 0.5, 0.099, 0.1}
		h, err := slowprovider.NewHandler(slowprovider.Options{
			Delay:           200 * time.Millisecond,
			TooManyRequests: 0.1,
			Rand:            func() float64 { r := rolls[0]; rolls = rolls[1:]; return r },
		})
		if err != nil {
			t.Fatal(err)
		}
		for i, want := range []int{http.StatusTooManyRequests, http.StatusOK, http.StatusTooManyRequests, http.StatusOK} {
			start := time.Now()
			rec := serve(t.Context(), h, http.MethodPost)
			if rec.Code != want {
				t.Errorf("request %d = %d, want %d", i, rec.Code, want)
			}
			if got := time.Since(start); got != 200*time.Millisecond {
				t.Errorf("request %d took %v, want 200ms", i, got)
			}
		}
	})
}

// 客戶端已經放棄（逾時、斷線）的請求，不再等、也不回應。
func TestHandlerStopsWaitingWhenTheClientGivesUp(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, err := slowprovider.NewHandler(slowprovider.Options{Delay: time.Hour})
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()
		start := time.Now()
		rec := serve(ctx, h, http.MethodPost)
		if got := time.Since(start); got != time.Second {
			t.Errorf("handler returned after %v, want 1s: the client's deadline", got)
		}
		if rec.Code != 0 {
			t.Errorf("handler answered %d to a client that had given up, want no response", rec.Code)
		}
	})
}

func TestHandlerOnlyAcceptsPostPush(t *testing.T) {
	t.Parallel()
	h, err := slowprovider.NewHandler(slowprovider.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if got := serve(t.Context(), h, http.MethodGet).Code; got != http.StatusMethodNotAllowed {
		t.Errorf("GET /push = %d, want 405", got)
	}
	if got := serve(t.Context(), h, http.MethodPost).Code; got != http.StatusOK {
		t.Errorf("POST /push with no delay and no 429 = %d, want 200", got)
	}
}

func TestNewHandlerRejectsInvalidOptions(t *testing.T) {
	t.Parallel()
	for name, opts := range map[string]slowprovider.Options{
		"negative delay":       {Delay: -time.Second},
		"share below zero":     {TooManyRequests: -0.1},
		"share above one":      {TooManyRequests: 1.1},
		"share is not a ratio": {TooManyRequests: 10},
	} {
		if _, err := slowprovider.NewHandler(opts); err == nil {
			t.Errorf("%s: NewHandler() error = nil, want an error", name)
		}
	}
}
