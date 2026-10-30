package rates_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"

	rates "github.com/holmeslin/learning-go-to-senior/examples/chapter10/02_retry_backoff_jitter/before"
)

// ANCHOR: example

// badRequest 對每次呼叫都回 400，並在第 100 次呼叫時取消 ctx：Fetch 自己不會停，
// 只有 ctx 能讓它停下來。
type badRequest struct {
	calls  int
	cancel context.CancelFunc
}

func (b *badRequest) RoundTrip(*http.Request) (*http.Response, error) {
	b.calls++
	if b.calls == 100 {
		b.cancel()
	}
	rec := httptest.NewRecorder()
	rec.WriteHeader(http.StatusBadRequest)
	return rec.Result(), nil
}

func Example() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	d := &badRequest{cancel: cancel}

	_, err := rates.Fetch(ctx, &http.Client{Transport: d}, target, 0)
	fmt.Println(d.calls, "calls for a request that can never succeed:", err)
	// Output:
	// 100 calls for a request that can never succeed: context canceled
}

// ANCHOR_END: example
