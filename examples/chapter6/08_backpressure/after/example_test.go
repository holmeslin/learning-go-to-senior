package thumbnail_test

import (
	"context"
	"fmt"
	"sync"

	thumbnail "github.com/holmeslin/learning-go-to-senior/examples/chapter6/08_backpressure/after"
)

// stall 是一個卡住的 consumer：收到第一筆後通知，然後等 release 關閉才繼續。
type stall struct {
	once    sync.Once
	started chan struct{}
	release chan struct{}
}

func newStall() *stall {
	return &stall{started: make(chan struct{}), release: make(chan struct{})}
}

func (s *stall) process(thumbnail.Job) {
	s.once.Do(func() { close(s.started) })
	<-s.release
}

// ANCHOR: example

func Example() {
	for _, n := range []int{100, 10000} {
		s := newStall()
		q := thumbnail.New(3, s.process)

		_ = q.TrySubmit(thumbnail.Job{Path: "photo-0.jpg"})
		<-s.started // consumer 已經取走第一筆，卡在縮圖上
		rejected := 0
		var lastErr error
		for range n - 1 {
			if err := q.TrySubmit(thumbnail.Job{Path: "photo.jpg"}); err != nil {
				rejected++
				lastErr = err
			}
		}
		fmt.Printf("submitted %d, pending %d, rejected %d (%v)\n", n, q.Pending(), rejected, lastErr)

		close(s.release)
		q.Close()
	}
	// Output:
	// submitted 100, pending 3, rejected 96 (queue full)
	// submitted 10000, pending 3, rejected 9996 (queue full)
}

// ANCHOR_END: example

// ANCHOR: wait

func Example_wait() {
	s := newStall()
	q := thumbnail.New(1, s.process)

	_ = q.TrySubmit(thumbnail.Job{Path: "a.jpg"})
	<-s.started
	_ = q.TrySubmit(thumbnail.Job{Path: "b.jpg"}) // 佇列滿了

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error)
	go func() { result <- q.Submit(ctx, thumbnail.Job{Path: "c.jpg"}) }()
	cancel() // consumer 卡住、佇列又滿，Submit 只能等到 ctx 結束
	fmt.Println("submit:", <-result)

	close(s.release)
	q.Close()
	// Output:
	// submit: context canceled
}

// ANCHOR_END: wait
