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

	bulkhead "github.com/holmeslin/learning-go-to-senior/examples/chapter10/08_bulkhead/after"
)

// roundTrip、replies、blocked 與 before 的測試一模一樣。

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func replies(body string, calls *atomic.Int32) roundTrip {
	return func(*http.Request) (*http.Response, error) {
		calls.Add(1)
		rec := httptest.NewRecorder()
		rec.WriteString(body)
		return rec.Result(), nil
	}
}

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

// ANCHOR: isolated

func TestSlowDownstreamDoesNotStarveOthers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var searchInflight, pricingCalls atomic.Int32
		release := make(chan struct{})
		defer close(release) // bubble 結束前，卡住的請求必須能離開
		g, err := bulkhead.New(map[string]bulkhead.Downstream{
			"search": {
				BaseURL: "http://search", Limit: 2,
				Client: &http.Client{Transport: blocked(release, &searchInflight)},
			},
			"pricing": {
				BaseURL: "http://pricing", Limit: 2,
				Client: &http.Client{Transport: replies("9.99", &pricingCalls)},
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		// search 卡住，它自己的兩個名額都被占住。
		for range 2 {
			go func() { _, _ = g.Get(context.Background(), "search", "/q") }()
		}
		synctest.Wait()

		// pricing 不受影響：立刻完成，假時間沒有前進。
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		start := time.Now()
		body, err := g.Get(ctx, "pricing", "/p")
		if err != nil || body != "9.99" {
			t.Errorf("Get(pricing) = %q, %v, want 9.99, nil", body, err)
		}
		if got := time.Since(start); got != 0 {
			t.Errorf("Get(pricing) took %v in the bubble, want 0", got)
		}

		// search 的第三個請求排不進去：下游同時只看到 2 個，它等到 deadline。
		third := make(chan error, 1)
		go func() { _, err := g.Get(ctx, "search", "/q"); third <- err }()
		synctest.Wait()
		if got := searchInflight.Load(); got != 2 {
			t.Errorf("search in flight = %d, want 2 (its limit)", got)
		}
		if err := <-third; !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("third Get(search) error = %v, want deadline exceeded", err)
		}
		if got := time.Since(start); got != time.Second {
			t.Errorf("third Get(search) gave up after %v in the bubble, want 1s", got)
		}
	})
}

// ANCHOR_END: isolated

func TestNewValidates(t *testing.T) {
	t.Parallel()
	_, err := bulkhead.New(map[string]bulkhead.Downstream{"x": {BaseURL: "http://x", Client: http.DefaultClient, Limit: 0}})
	if !errors.Is(err, bulkhead.ErrInvalidLimit) {
		t.Errorf("New(limit 0) error = %v, want ErrInvalidLimit", err)
	}
	g, err := bulkhead.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.Get(context.Background(), "nope", "/"); !errors.Is(err, bulkhead.ErrUnknownDownstream) {
		t.Errorf("Get(nope) error = %v, want ErrUnknownDownstream", err)
	}
}

func TestGetReportsNon200(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	g, err := bulkhead.New(map[string]bulkhead.Downstream{"x": {BaseURL: srv.URL, Client: srv.Client(), Limit: 1}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.Get(context.Background(), "x", "/"); err == nil {
		t.Error("Get() error = nil on 404, want error")
	}
}
