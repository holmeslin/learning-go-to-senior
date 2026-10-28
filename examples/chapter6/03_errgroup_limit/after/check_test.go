package addrcheck_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	addrcheck "github.com/holmeslin/learning-go-to-senior/examples/chapter6/03_errgroup_limit/after"
)

// counter 記錄同時進行中的呼叫數與它的最大值。
type counter struct {
	mu       sync.Mutex
	inflight int
	peak     int
}

func (c *counter) enter() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.inflight++
	c.peak = max(c.peak, c.inflight)
}

func (c *counter) leave() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.inflight--
}

// Peak 回傳同時進行中的呼叫數曾經達到的最大值。
func (c *counter) Peak() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.peak
}

// ANCHOR: limit

func TestCheckAllLimit(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var c counter
		check := func(ctx context.Context, addr string) error {
			c.enter()
			defer c.leave()
			time.Sleep(100 * time.Millisecond) // bubble 內的假時鐘，不會真的等
			return nil
		}

		start := time.Now()
		err := addrcheck.CheckAll(t.Context(), make([]string, 12), 5, check)
		if err != nil {
			t.Fatalf("CheckAll() = %v, want nil", err)
		}
		if got, want := c.Peak(), 5; got != want {
			t.Errorf("peak concurrent calls = %d, want %d", got, want)
		}
		// 12 筆、一次 5 筆：5 + 5 + 2，三輪。
		if got, want := time.Since(start), 300*time.Millisecond; got != want {
			t.Errorf("CheckAll() took %v, want %v", got, want)
		}
	})
}

// ANCHOR_END: limit

func TestCheckAllReturnsFirstError(t *testing.T) {
	errBad := errors.New("bad address")
	check := func(ctx context.Context, addr string) error {
		if addr == "bad" {
			return errBad
		}
		return nil
	}

	err := addrcheck.CheckAll(t.Context(), []string{"a", "bad", "b", "c"}, 2, check)
	if !errors.Is(err, errBad) {
		t.Errorf("CheckAll() = %v, want %v", err, errBad)
	}
}

func TestCheckAllInvalidLimit(t *testing.T) {
	check := func(ctx context.Context, addr string) error {
		t.Error("check must not be called with an invalid limit")
		return nil
	}

	for _, limit := range []int{0, -1} {
		// 沒有檢查的話，limit 0 會讓第一個 Go 永遠等下去。
		err := addrcheck.CheckAll(t.Context(), []string{"a"}, limit, check)
		if !errors.Is(err, addrcheck.ErrInvalidLimit) {
			t.Errorf("CheckAll(limit=%d) = %v, want %v", limit, err, addrcheck.ErrInvalidLimit)
		}
	}
}
