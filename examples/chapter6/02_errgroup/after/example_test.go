package fetch_test

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	fetch "github.com/holmeslin/learning-go-to-senior/examples/chapter6/02_errgroup/after"
)

// ANCHOR: example

func ExampleFetchAll() {
	var canceled atomic.Int32
	get := func(ctx context.Context, src string) error {
		if src == "b" {
			return errors.New("fetch b: not found")
		}
		<-ctx.Done() // 其他來源要抓很久，只有被取消才會結束
		canceled.Add(1)
		return ctx.Err()
	}

	err := fetch.FetchAll(context.Background(), []string{"a", "b", "c", "d", "e"}, get)
	fmt.Println("error:", err)
	fmt.Println("canceled:", canceled.Load())
	// Output:
	// error: fetch b: not found
	// canceled: 4
}

// ANCHOR_END: example

// ANCHOR: joinexample

func ExampleFetchAllJoin() {
	get := func(ctx context.Context, src string) error {
		switch src {
		case "a":
			return errors.New("fetch a: not found")
		case "c":
			return errors.New("fetch c: timeout")
		}
		return nil
	}

	err := fetch.FetchAllJoin(context.Background(), []string{"a", "b", "c"}, get)
	fmt.Println(err)
	// Output:
	// fetch a: not found
	// fetch c: timeout
}

// ANCHOR_END: joinexample
