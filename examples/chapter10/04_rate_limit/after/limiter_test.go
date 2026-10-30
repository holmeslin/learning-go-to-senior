package ratelimit_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"golang.org/x/time/rate"

	ratelimit "github.com/holmeslin/learning-go-to-senior/examples/chapter10/04_rate_limit/after"
)

// ANCHOR: helpers

// ok 是被保護的 handler：永遠回 200。
var ok = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

// serve 以 client 的身分對 h 發一個請求。
func serve(h http.Handler, client string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Client-ID", client)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// ANCHOR_END: helpers

// ANCHOR: wait

func TestThrottleSpacesConcurrentCalls(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var mu sync.Mutex
		var at []time.Duration // 每次呼叫下游的時間，距離開始多久
		start := time.Now()
		send := ratelimit.Throttle(rate.NewLimiter(5, 5), func(context.Context, string) error {
			mu.Lock()
			defer mu.Unlock()
			at = append(at, time.Since(start))
			return nil
		})

		var wg sync.WaitGroup
		for range 10 {
			wg.Go(func() {
				if err := send(t.Context(), "hello"); err != nil {
					t.Errorf("send() = %v, want nil", err)
				}
			})
		}
		wg.Wait()

		slices.Sort(at)
		ms := time.Millisecond
		want := []time.Duration{0, 0, 0, 0, 0, 200 * ms, 400 * ms, 600 * ms, 800 * ms, 1000 * ms}
		if !slices.Equal(at, want) {
			t.Errorf("downstream called at %v, want %v", at, want)
		}
	})
}

// ANCHOR_END: wait

// ANCHOR: deadline

func TestThrottleGivesUpWhenTokenComesAfterDeadline(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		calls := 0
		send := ratelimit.Throttle(rate.NewLimiter(rate.Every(time.Second), 1), func(context.Context, string) error {
			calls++
			return nil
		})
		if err := send(t.Context(), "first"); err != nil {
			t.Fatalf("first send() = %v, want nil", err)
		}

		start := time.Now()
		ctx, cancel := context.WithTimeout(t.Context(), 500*time.Millisecond)
		defer cancel()
		if err := send(ctx, "second"); err == nil {
			t.Error("second send() = nil, want an error: the next token is 1s away, the deadline 500ms")
		}
		if got := time.Since(start); got != 0 {
			t.Errorf("second send() waited %v, want 0: it should give up at once", got)
		}
		if calls != 1 {
			t.Errorf("downstream called %d times, want 1", calls)
		}

		ctx, cancel = context.WithTimeout(t.Context(), 2*time.Second)
		defer cancel()
		if err := send(ctx, "third"); err != nil {
			t.Errorf("third send() = %v, want nil", err)
		}
		if got := time.Since(start); got != time.Second {
			t.Errorf("third send() waited %v, want 1s", got)
		}
	})
}

// ANCHOR_END: deadline

// ANCHOR: clients

func TestPerClientRejectsOnlyTheClientOverTheLimit(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := ratelimit.NewPerClient(1, 3).Handler(ok) // 每秒補 1 個，突發最多 3 個

		var codes []int
		for range 5 {
			rec := serve(h, "alice")
			codes = append(codes, rec.Code)
			if rec.Code == http.StatusTooManyRequests && rec.Header().Get("Retry-After") != "1" {
				t.Errorf("Retry-After = %q, want 1", rec.Header().Get("Retry-After"))
			}
		}
		if want := []int{200, 200, 200, 429, 429}; !slices.Equal(codes, want) {
			t.Errorf("alice got %v, want %v", codes, want)
		}
		if got := serve(h, "bob").Code; got != http.StatusOK {
			t.Errorf("bob got %d while alice is limited, want 200", got)
		}

		time.Sleep(time.Second) // 假時間：補回 1 個 token
		if got := serve(h, "alice").Code; got != http.StatusOK {
			t.Errorf("alice after 1s got %d, want 200", got)
		}
		if got := serve(h, "alice").Code; got != http.StatusTooManyRequests {
			t.Errorf("alice's second request after 1s got %d, want 429", got)
		}
	})
}

// ANCHOR_END: clients

// ANCHOR: retryafter

func TestRetryAfterCountsDown(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := ratelimit.NewPerClient(rate.Every(10*time.Second), 1).Handler(ok) // 每 10 秒補 1 個
		start := time.Now()
		check := func(wantCode int, wantRetryAfter string) {
			t.Helper()
			rec := serve(h, "alice")
			if got := rec.Header().Get("Retry-After"); rec.Code != wantCode || got != wantRetryAfter {
				t.Errorf("at %v: got %d Retry-After %q, want %d %q", time.Since(start), rec.Code, got, wantCode, wantRetryAfter)
			}
		}

		check(200, "")
		check(429, "10")
		time.Sleep(4 * time.Second)
		check(429, "6")
		time.Sleep(6 * time.Second)
		check(200, "")
	})
}

// ANCHOR_END: retryafter
