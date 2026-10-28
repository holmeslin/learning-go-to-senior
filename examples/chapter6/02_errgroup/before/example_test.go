package fetch_test

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	fetch "github.com/holmeslin/learning-go-to-senior/examples/chapter6/02_errgroup/before"
)

// ANCHOR: example

func ExampleFetchAll() {
	failed := make(chan struct{})
	var finished atomic.Int32
	get := func(ctx context.Context, src string) error {
		if src == "b" {
			close(failed)
			return errors.New("fetch b: not found")
		}
		select {
		case <-failed: // b 已經出錯了，其他來源照樣抓完
			finished.Add(1)
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	err := fetch.FetchAll(context.Background(), []string{"a", "b", "c", "d", "e"}, get)
	fmt.Println("error:", err)
	fmt.Println("finished:", finished.Load())
	// Output:
	// error: fetch b: not found
	// finished: 4
}

// ANCHOR_END: example
