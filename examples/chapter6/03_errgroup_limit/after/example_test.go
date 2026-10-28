package addrcheck_test

import (
	"context"
	"fmt"

	"golang.org/x/sync/errgroup"

	addrcheck "github.com/holmeslin/learning-go-to-senior/examples/chapter6/03_errgroup_limit/after"
)

// ANCHOR: example

func ExampleCheckAll() {
	addrs := make([]string, 20)
	var c counter
	entered := make(chan struct{}, len(addrs))
	release := make(chan struct{})
	check := func(ctx context.Context, addr string) error {
		c.enter()
		defer c.leave()
		entered <- struct{}{}
		<-release
		return nil
	}

	done := make(chan error)
	go func() { done <- addrcheck.CheckAll(context.Background(), addrs, 5, check) }()

	for range 5 {
		<-entered // 上限 5：先等前 5 筆進了下游，才放行
	}
	close(release)
	if err := <-done; err != nil {
		fmt.Println("error:", err)
	}
	fmt.Println("addresses:", len(addrs))
	fmt.Println("peak concurrent calls:", c.Peak())
	// Output:
	// addresses: 20
	// peak concurrent calls: 5
}

// ANCHOR_END: example

// ANCHOR: trygo

func Example_tryGo() {
	var g errgroup.Group
	g.SetLimit(2)

	release := make(chan struct{})
	block := func() error { <-release; return nil }
	for i := 1; i <= 3; i++ {
		fmt.Printf("task %d started: %v\n", i, g.TryGo(block))
	}
	close(release)
	_ = g.Wait()
	// Output:
	// task 1 started: true
	// task 2 started: true
	// task 3 started: false
}

// ANCHOR_END: trygo
