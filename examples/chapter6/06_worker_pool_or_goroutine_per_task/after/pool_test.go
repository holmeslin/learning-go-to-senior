package batch_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	batch "github.com/holmeslin/learning-go-to-senior/examples/chapter6/06_worker_pool_or_goroutine_per_task/after"
)

// conn 是測試用的連線。handled 沒有加鎖：一條連線只屬於一個 worker，
// 如果有兩個 goroutine 碰它，-race 會報告。
type conn struct {
	handled int
	stats   *stats
}

// stats 統計建立、關閉的連線數與處理過的工作數。
type stats struct{ dials, closes, handled atomic.Int32 }

func (c *conn) Close() error {
	c.stats.closes.Add(1)
	c.stats.handled.Add(int32(c.handled))
	return nil
}

// dial 回傳建立連線的函式，並記下建立了幾條。
func (s *stats) dial() (*conn, error) {
	s.dials.Add(1)
	return &conn{stats: s}, nil
}

func handle(ctx context.Context, c *conn, job int) error {
	c.handled++
	return nil
}

// ANCHOR: once

func TestConsumeDialsOncePerWorker(t *testing.T) {
	var s stats
	jobs := make(chan int, 100)
	for i := range 100 {
		jobs <- i
	}
	close(jobs)

	if err := batch.Consume(t.Context(), 3, s.dial, jobs, handle); err != nil {
		t.Fatalf("Consume() error = %v", err)
	}
	if got := s.dials.Load(); got != 3 {
		t.Errorf("dials = %d, want 3 (one per worker, not one per job)", got)
	}
	if got := s.closes.Load(); got != 3 {
		t.Errorf("closes = %d, want 3", got)
	}
	if got := s.handled.Load(); got != 100 {
		t.Errorf("handled = %d, want 100", got)
	}
}

// ANCHOR_END: once

// ANCHOR: endless

func TestConsumeStopsWhenContextEnds(t *testing.T) {
	var s stats
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	jobs := make(chan int) // 沒有盡頭的來源：不會 close，只有 ctx 能讓它停
	producerDone := make(chan struct{})
	go func() {
		defer close(producerDone)
		for i := 0; ; i++ {
			select {
			case jobs <- i:
			case <-ctx.Done():
				return
			}
		}
	}()

	var count atomic.Int32
	err := batch.Consume(ctx, 3, s.dial, jobs, func(ctx context.Context, c *conn, job int) error {
		if count.Add(1) == 20 {
			cancel() // 處理到第 20 個工作時結束，不用靠時間
		}
		return handle(ctx, c, job)
	})
	<-producerDone // 確認 producer 也停了，測試結束時沒有 goroutine 留下

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Consume() error = %v, want %v", err, context.Canceled)
	}
	if dials, closes := s.dials.Load(), s.closes.Load(); closes != dials {
		t.Errorf("closes = %d, want %d (every dialed conn is closed)", closes, dials)
	}
	if got := s.handled.Load(); got < 20 {
		t.Errorf("handled = %d, want at least 20", got)
	}
}

// ANCHOR_END: endless

func TestConsumeDialError(t *testing.T) {
	var s stats
	errDial := errors.New("dial failed")
	dial := func() (*conn, error) {
		if s.dials.Add(1) == 2 {
			return nil, errDial
		}
		return &conn{stats: &s}, nil
	}

	err := batch.Consume(t.Context(), 2, dial, make(chan int), handle)
	if !errors.Is(err, errDial) {
		t.Errorf("Consume() error = %v, want %v", err, errDial)
	}
	if got := s.closes.Load(); got != 1 {
		t.Errorf("closes = %d, want 1 (the conn that was dialed)", got)
	}
}
