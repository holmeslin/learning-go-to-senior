package thumbnail_test

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	thumbnail "github.com/holmeslin/learning-go-to-senior/examples/chapter6/08_backpressure/after"
)

const capacity = 3

// gatedQueue 建立一個 consumer 卡住的佇列：process 每收到一筆就等 gate 一個訊號。
// 要在 synctest bubble 裡呼叫；回傳時 consumer 已取走第一筆，佇列是空的。
func gatedQueue(t *testing.T) (q *thumbnail.Queue, gate chan struct{}) {
	t.Helper()
	gate = make(chan struct{})
	q = thumbnail.New(capacity, func(thumbnail.Job) { <-gate })
	t.Cleanup(func() { close(gate); q.Close() })

	if err := q.Submit(t.Context(), thumbnail.Job{Path: "first.jpg"}); err != nil {
		t.Fatalf("Submit() = %v, want nil", err)
	}
	synctest.Wait() // consumer 取走第一筆、卡在 gate
	return q, gate
}

func fill(t *testing.T, q *thumbnail.Queue) {
	t.Helper()
	for range capacity {
		if err := q.TrySubmit(thumbnail.Job{Path: "queued.jpg"}); err != nil {
			t.Fatalf("TrySubmit() = %v, want nil", err)
		}
	}
}

// ANCHOR: wait

func TestSubmitWaitsForRoom(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		q, gate := gatedQueue(t)
		fill(t, q)

		submitted := make(chan error, 1)
		go func() { submitted <- q.Submit(t.Context(), thumbnail.Job{Path: "late.jpg"}) }()
		synctest.Wait() // producer 已經卡在送出
		select {
		case err := <-submitted:
			t.Fatalf("Submit() returned %v while the queue is full, want it to wait", err)
		default:
		}

		gate <- struct{}{} // consumer 做完一筆、取下一筆，空出一格
		synctest.Wait()
		select {
		case err := <-submitted:
			if err != nil {
				t.Errorf("Submit() = %v, want nil", err)
			}
		default:
			t.Error("Submit() still waiting after a slot opened")
		}
	})
}

// ANCHOR_END: wait

func TestSubmitGivesUpWhenContextEnds(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		q, _ := gatedQueue(t)
		fill(t, q)

		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()
		start := time.Now()
		err := q.Submit(ctx, thumbnail.Job{Path: "late.jpg"})

		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("Submit() = %v, want %v", err, context.DeadlineExceeded)
		}
		// bubble 內的假時鐘：等到 deadline 為止，不多也不少。
		if got, want := time.Since(start), time.Second; got != want {
			t.Errorf("Submit() waited %v, want %v", got, want)
		}
		if got := q.Pending(); got != capacity {
			t.Errorf("Pending() = %d, want %d", got, capacity)
		}
	})
}

// ANCHOR: reject

func TestTrySubmitRejectsWhenFull(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		q, gate := gatedQueue(t)
		fill(t, q)

		err := q.TrySubmit(thumbnail.Job{Path: "extra.jpg"})
		if !errors.Is(err, thumbnail.ErrFull) {
			t.Errorf("TrySubmit() on a full queue = %v, want %v", err, thumbnail.ErrFull)
		}
		if got := q.Pending(); got != capacity {
			t.Errorf("Pending() = %d, want %d: a rejected job must not be queued", got, capacity)
		}

		gate <- struct{}{} // 空出一格之後又收得下
		synctest.Wait()
		if err := q.TrySubmit(thumbnail.Job{Path: "retry.jpg"}); err != nil {
			t.Errorf("TrySubmit() after a slot opened = %v, want nil", err)
		}
	})
}

// ANCHOR_END: reject

func TestSubmitNeverExceedsCapacity(t *testing.T) {
	const total = 1000
	processed := 0
	q := thumbnail.New(capacity, func(thumbnail.Job) { processed++ })

	peak := 0
	for range total {
		if err := q.Submit(t.Context(), thumbnail.Job{Path: "photo.jpg"}); err != nil {
			t.Fatalf("Submit() = %v, want nil", err)
		}
		peak = max(peak, q.Pending())
	}
	q.Close() // 等 consumer 做完；processed 只由 consumer 寫入，Close 之後才讀

	if peak > capacity {
		t.Errorf("peak pending = %d, want at most %d", peak, capacity)
	}
	if processed != total {
		t.Errorf("processed %d jobs, want %d", processed, total)
	}
}
