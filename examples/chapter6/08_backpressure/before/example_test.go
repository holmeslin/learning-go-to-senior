package thumbnail_test

import (
	"fmt"
	"sync"

	thumbnail "github.com/holmeslin/learning-go-to-senior/examples/chapter6/08_backpressure/before"
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
		q := thumbnail.New(s.process)

		q.Submit(thumbnail.Job{Path: "photo-0.jpg"})
		<-s.started // consumer 已經取走第一筆，卡在縮圖上
		for range n - 1 {
			q.Submit(thumbnail.Job{Path: "photo.jpg"})
		}
		fmt.Printf("submitted %d, pending %d\n", n, q.Pending())

		close(s.release)
		q.Close()
	}
	// Output:
	// submitted 100, pending 99
	// submitted 10000, pending 9999
}

// ANCHOR_END: example
