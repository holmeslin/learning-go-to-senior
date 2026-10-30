package bulkhead_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	bulkhead "github.com/holmeslin/learning-go-to-senior/examples/chapter10/08_bulkhead/before"
)

// ANCHOR: stubs

// roundTrip 讓一個函式充當 http.RoundTripper，測試用它取代網路。
type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// replies 立刻回 200 與 body，並記下被呼叫的次數。
func replies(body string, calls *atomic.Int32) roundTrip {
	return func(*http.Request) (*http.Response, error) {
		calls.Add(1)
		rec := httptest.NewRecorder()
		rec.WriteString(body)
		return rec.Result(), nil
	}
}

// blocked 模擬卡住的下游：請求進來就算一個進行中，直到 release 關閉或請求的 context 結束。
func blocked(release <-chan struct{}, inflight *atomic.Int32) roundTrip {
	return func(r *http.Request) (*http.Response, error) {
		inflight.Add(1)
		defer inflight.Add(-1)
		select {
		case <-release:
			return httptest.NewRecorder().Result(), nil
		case <-r.Context().Done():
			return nil, r.Context().Err()
		}
	}
}

// byHost 依請求的 host 轉給不同的下游。
func byHost(m map[string]roundTrip) roundTrip {
	return func(r *http.Request) (*http.Response, error) { return m[r.URL.Host](r) }
}

// ANCHOR_END: stubs

// ANCHOR: starved

func TestSlowDownstreamStarvesOthers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var searchInflight, pricingCalls atomic.Int32
		release := make(chan struct{})
		defer close(release) // bubble 結束前，卡住的請求必須能離開
		client := &http.Client{Transport: byHost(map[string]roundTrip{
			"search":  blocked(release, &searchInflight),
			"pricing": replies("9.99", &pricingCalls),
		})}
		g, err := bulkhead.New(2, client, map[string]string{
			"search": "http://search", "pricing": "http://pricing",
		})
		if err != nil {
			t.Fatal(err)
		}

		// search 卡住，兩個 worker 都被它占住。
		for range 2 {
			go g.Get(context.Background(), "search", "/q")
		}
		synctest.Wait()
		if got := searchInflight.Load(); got != 2 {
			t.Fatalf("search in flight = %d, want 2", got)
		}

		// pricing 本身是好的，卻拿不到 worker，直到呼叫端的 deadline。
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		start := time.Now()
		_, err = g.Get(ctx, "pricing", "/p")

		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("Get(pricing) error = %v, want deadline exceeded", err)
		}
		if got := time.Since(start); got != time.Second {
			t.Errorf("Get(pricing) took %v in the bubble, want 1s", got)
		}
		if got := pricingCalls.Load(); got != 0 {
			t.Errorf("pricing was called %d times, want 0 (it never got a worker)", got)
		}
	})
}

// ANCHOR_END: starved

func TestGetHealthy(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	g, err := bulkhead.New(1, &http.Client{Transport: replies("9.99", &calls)}, map[string]string{"pricing": "http://pricing"})
	if err != nil {
		t.Fatal(err)
	}
	if body, err := g.Get(context.Background(), "pricing", "/p"); err != nil || body != "9.99" {
		t.Errorf("Get(pricing) = %q, %v, want 9.99, nil", body, err)
	}
	if _, err := g.Get(context.Background(), "nope", "/"); err == nil {
		t.Error("Get(nope) error = nil, want unknown downstream")
	}
	if _, err := bulkhead.New(0, http.DefaultClient, nil); err == nil {
		t.Error("New(0, ...) error = nil, want error")
	}
}
