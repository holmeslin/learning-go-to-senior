package batch_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	batch "github.com/holmeslin/learning-go-to-senior/examples/chapter6/06_worker_pool_or_goroutine_per_task/after"
)

func ids(n int) []int {
	out := make([]int, n)
	for i := range out {
		out[i] = i
	}
	return out
}

func TestFetchAllKeepsOrder(t *testing.T) {
	fetch := func(ctx context.Context, id int) (string, error) {
		return fmt.Sprintf("page-%d", id), nil
	}
	got, err := batch.FetchAll(t.Context(), ids(20), 4, fetch)
	if err != nil {
		t.Fatalf("FetchAll() error = %v", err)
	}
	want := make([]string, 20)
	for i := range want {
		want[i] = fmt.Sprintf("page-%d", i)
	}
	if !slices.Equal(got, want) {
		t.Errorf("FetchAll() = %v, want %v", got, want)
	}
}

// ANCHOR: limit

func TestFetchAllLimit(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var running, peak atomic.Int32
		fetch := func(ctx context.Context, id int) (string, error) {
			n := running.Add(1)
			defer running.Add(-1)
			for { // 記下同時執行數的最大值
				p := peak.Load()
				if n <= p || peak.CompareAndSwap(p, n) {
					break
				}
			}
			time.Sleep(time.Second) // bubble 裡的假時間，不是真的等
			return "", nil
		}
		start := time.Now()
		if _, err := batch.FetchAll(t.Context(), ids(7), 3, fetch); err != nil {
			t.Fatalf("FetchAll() error = %v", err)
		}
		if got := peak.Load(); got != 3 {
			t.Errorf("peak concurrency = %d, want 3", got)
		}
		if got, want := time.Since(start), 3*time.Second; got != want {
			t.Errorf("elapsed = %v, want %v (7 tasks, 3 at a time)", got, want)
		}
	})
}

// ANCHOR_END: limit

// ANCHOR: firsterror

func TestFetchAllReturnsFirstError(t *testing.T) {
	errBoom := errors.New("boom")
	fetch := func(ctx context.Context, id int) (string, error) {
		if id == 0 {
			return "", errBoom
		}
		<-ctx.Done() // 其他工作等著被取消，才不用靠時間
		return "", ctx.Err()
	}
	got, err := batch.FetchAll(t.Context(), ids(3), 3, fetch)
	if !errors.Is(err, errBoom) {
		t.Errorf("FetchAll() error = %v, want %v", err, errBoom)
	}
	if got != nil {
		t.Errorf("FetchAll() = %v, want nil on error", got)
	}
}

// ANCHOR_END: firsterror

func TestFetchAllRejectsBadLimit(t *testing.T) {
	fetch := func(ctx context.Context, id int) (string, error) {
		t.Error("fetch must not be called")
		return "", nil
	}
	for _, limit := range []int{0, -1} {
		if _, err := batch.FetchAll(t.Context(), ids(3), limit, fetch); !errors.Is(err, batch.ErrInvalidLimit) {
			t.Errorf("FetchAll(limit=%d) error = %v, want %v", limit, err, batch.ErrInvalidLimit)
		}
	}
}
