package addrcheck_test

import (
	"context"
	"fmt"
	"sync"

	addrcheck "github.com/holmeslin/learning-go-to-senior/examples/chapter6/03_errgroup_limit/before"
)

// downstream 記錄同時進行中的呼叫數與它的最大值。
type downstream struct {
	mu       sync.Mutex
	inflight int
	peak     int
	entered  chan struct{} // 每個呼叫進來時送一個訊號
	release  chan struct{} // 關閉後，卡住的呼叫才會回傳
}

func (d *downstream) check(ctx context.Context, addr string) error {
	d.mu.Lock()
	d.inflight++
	d.peak = max(d.peak, d.inflight)
	d.mu.Unlock()

	d.entered <- struct{}{}
	<-d.release

	d.mu.Lock()
	d.inflight--
	d.mu.Unlock()
	return nil
}

// ANCHOR: example

func ExampleCheckAll() {
	addrs := make([]string, 20)
	d := &downstream{entered: make(chan struct{}, len(addrs)), release: make(chan struct{})}

	done := make(chan error)
	go func() { done <- addrcheck.CheckAll(context.Background(), addrs, d.check) }()

	for range addrs {
		<-d.entered // 等到 20 筆都進了下游，才放行
	}
	close(d.release)
	if err := <-done; err != nil {
		fmt.Println("error:", err)
	}
	fmt.Println("addresses:", len(addrs))
	fmt.Println("peak concurrent calls:", d.peak)
	// Output:
	// addresses: 20
	// peak concurrent calls: 20
}

// ANCHOR_END: example
