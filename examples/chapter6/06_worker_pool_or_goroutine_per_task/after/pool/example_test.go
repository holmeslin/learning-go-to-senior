package pool_test

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter6/06_worker_pool_or_goroutine_per_task/after/pool"
)

// ANCHOR: consume

// session 是 Example 用的連線，Close 只是什麼都不做。
type session struct{}

func (session) Close() error { return nil }

func ExampleConsume() {
	var dials, handled atomic.Int32
	dial := func() (session, error) { dials.Add(1); return session{}, nil }
	work := func(ctx context.Context, s session, job int) error { handled.Add(1); return nil }

	jobs := make(chan int, 10)
	for i := range 10 {
		jobs <- i
	}
	close(jobs)

	err := pool.Consume(context.Background(), 3, dial, jobs, work)
	fmt.Printf("dials=%d handled=%d err=%v\n", dials.Load(), handled.Load(), err)
	// Output:
	// dials=3 handled=10 err=<nil>
}

// ANCHOR_END: consume
