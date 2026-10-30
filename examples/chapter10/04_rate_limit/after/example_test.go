package ratelimit_test

import (
	"fmt"
	"time"

	"golang.org/x/time/rate"

	ratelimit "github.com/holmeslin/learning-go-to-senior/examples/chapter10/04_rate_limit/after"
)

// ANCHOR: example

func Example() {
	// 每小時補 1 個、突發最多 2 個，所以這個 Example 不必等任何時間。
	h := ratelimit.NewPerClient(rate.Every(time.Hour), 2).Handler(ok)

	for i := 1; i <= 3; i++ {
		rec := serve(h, "alice")
		fmt.Printf("alice #%d: %d Retry-After=%q\n", i, rec.Code, rec.Header().Get("Retry-After"))
	}
	fmt.Println("bob #1:", serve(h, "bob").Code)
	// Output:
	// alice #1: 200 Retry-After=""
	// alice #2: 200 Retry-After=""
	// alice #3: 429 Retry-After="3600"
	// bob #1: 200
}

// ANCHOR_END: example
