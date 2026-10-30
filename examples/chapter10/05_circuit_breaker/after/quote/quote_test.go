package quote_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/synctest"
	"time"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter10/05_circuit_breaker/after/breaker"
	quote "github.com/holmeslin/learning-go-to-senior/examples/chapter10/05_circuit_breaker/after/quote"
)

// ANCHOR: failfast

func TestOpenBreakerFailsFast(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		calls, carrierUp := 0, false
		carrier := func(ctx context.Context) (int, error) {
			calls++
			if carrierUp {
				return 120, nil
			}
			<-ctx.Done() // 下游掛了：不回應，只等到逾時
			return 0, ctx.Err()
		}
		b := breaker.New(breaker.Config{Threshold: 3, Cooldown: 30 * time.Second})
		h := quote.Handler{Downstream: carrier, Breaker: b, Timeout: 3 * time.Second, RetryAfter: 30 * time.Second}
		get := func() *httptest.ResponseRecorder {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/quote", nil))
			return rec
		}

		start := time.Now()
		var codes []int
		for range 10 {
			rec := get()
			codes = append(codes, rec.Code)
			if rec.Code == http.StatusServiceUnavailable && rec.Header().Get("Retry-After") != "30" {
				t.Errorf("Retry-After = %q, want 30", rec.Header().Get("Retry-After"))
			}
		}

		// 前 3 個請求各等 3 秒逾時；斷路器打開之後的 7 個，立刻被擋下。
		if calls != 3 {
			t.Errorf("downstream calls = %d, want 3", calls)
		}
		if got := time.Since(start); got != 9*time.Second {
			t.Errorf("10 requests took %v in the bubble, want 9s", got)
		}
		for i, code := range codes {
			want := http.StatusServiceUnavailable
			if i < 3 {
				want = http.StatusGatewayTimeout
			}
			if code != want {
				t.Errorf("request %d = %d, want %d", i, code, want)
			}
		}

		// 冷卻結束、下游恢復：試探成功，之後的請求恢復正常。
		carrierUp = true
		time.Sleep(30 * time.Second) // synctest 的假時間
		for i := range 2 {
			if rec := get(); rec.Code != http.StatusOK || rec.Body.String() != "120" {
				t.Errorf("request %d after recovery = %d %q, want 200 \"120\"", i, rec.Code, rec.Body.String())
			}
		}
		if b.State() != breaker.Closed {
			t.Errorf("state = %v, want closed", b.State())
		}
	})
}

// ANCHOR_END: failfast

func TestRetryAfterRoundsUpToWholeSeconds(t *testing.T) {
	for _, tc := range []struct {
		after time.Duration
		want  string
	}{
		{0, "1"}, // 標頭的最小單位是秒，0 秒等於叫呼叫端立刻重試
		{500 * time.Millisecond, "1"},
		{time.Second, "1"},
		{1500 * time.Millisecond, "2"},
		{30 * time.Second, "30"},
	} {
		h := quote.Handler{
			Downstream: func(context.Context) (int, error) { return 0, errors.New("down") },
			Breaker:    breaker.New(breaker.Config{Threshold: 1, Cooldown: time.Hour}),
			Timeout:    time.Second,
			RetryAfter: tc.after,
		}
		get := func() *httptest.ResponseRecorder {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/quote", nil))
			return rec
		}

		get() // 第一個失敗，斷路器打開
		rec := get()

		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("RetryAfter %v: status = %d, want 503", tc.after, rec.Code)
		}
		if got := rec.Header().Get("Retry-After"); got != tc.want {
			t.Errorf("RetryAfter %v: Retry-After = %q, want %q", tc.after, got, tc.want)
		}
	}
}
