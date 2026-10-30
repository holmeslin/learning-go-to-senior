package rates_test

import (
	"context"
	"fmt"
	"net/http"

	rates "github.com/holmeslin/learning-go-to-senior/examples/chapter10/02_retry_backoff_jitter/after"
)

// ANCHOR: example

func Example() {
	// Base 是 0，等待時間就是 0，這裡不需要等真的時間。
	policy := rates.Policy{MaxAttempts: 3}

	d := newDownstream(failFirst(2, status(http.StatusServiceUnavailable, "")))
	body, err := client(d, policy).Fetch(context.Background(), target)
	fmt.Printf("503 twice: %s, %v, %d calls\n", body, err, len(d.at))

	d = newDownstream(status(http.StatusBadRequest, ""))
	_, err = client(d, policy).Fetch(context.Background(), target)
	fmt.Printf("400: %v, %d call\n", err, len(d.at))
	// Output:
	// 503 twice: ok, <nil>, 3 calls
	// 400: unexpected status 400, 1 call
}

// ANCHOR_END: example
