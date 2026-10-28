package sourceread_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"testing/synctest"

	sourceread "github.com/holmeslin/learning-go-to-senior/examples/chapter8/07_blocking_syscall_scheduling/after"
	before "github.com/holmeslin/learning-go-to-senior/examples/chapter8/07_blocking_syscall_scheduling/before"
)

// ANCHOR: counter

// counter 記錄同時進行中的讀取數與它的最大值。
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

// Inflight 回傳目前同時進行中的讀取數。
func (c *counter) Inflight() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.inflight
}

// Peak 回傳同時進行中的讀取數曾經達到的最大值。
func (c *counter) Peak() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.peak
}

// ANCHOR_END: counter

// ANCHOR: limit

func TestReadAllLimit(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var c counter
		release := make(chan struct{})
		read := func(ctx context.Context, n int) (int, error) {
			c.enter()
			defer c.leave()
			<-release // 真的阻塞，直到測試放行
			return n, nil
		}

		sources := make([]int, 20)
		for i := range sources {
			sources[i] = i
		}
		done := make(chan int)
		go func() {
			total, _ := sourceread.ReadAll(t.Context(), sources, 4, read)
			done <- total
		}()

		synctest.Wait() // 全部卡住了：上限內的 4 個在讀，ReadAll 卡在等空位
		if got, want := c.Inflight(), 4; got != want {
			t.Errorf("reads in flight while blocked = %d, want %d", got, want)
		}
		close(release)
		if got, want := <-done, 190; got != want { // 0 + 1 + ... + 19
			t.Errorf("ReadAll() total = %d, want %d", got, want)
		}
		if got, want := c.Peak(), 4; got != want {
			t.Errorf("peak concurrent reads = %d, want %d", got, want)
		}
	})
}

// ANCHOR_END: limit

func TestReadAllSameResultAsBefore(t *testing.T) {
	sources := []int{3, 1, 4, 1, 5, 9, 2, 6}
	read := func(ctx context.Context, n int) (int, error) { return n * 10, nil }

	want, err := before.ReadAll(t.Context(), sources, read)
	if err != nil {
		t.Fatalf("before.ReadAll() error = %v", err)
	}
	got, err := sourceread.ReadAll(t.Context(), sources, 3, read)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if got != want {
		t.Errorf("ReadAll() = %d, want %d (same as before)", got, want)
	}
}

func TestReadAllReturnsFirstError(t *testing.T) {
	errBad := errors.New("bad source")
	read := func(ctx context.Context, n int) (int, error) {
		if n == 2 {
			return 0, errBad
		}
		return n, nil
	}

	_, err := sourceread.ReadAll(t.Context(), []int{1, 2, 3, 4}, 2, read)
	if !errors.Is(err, errBad) {
		t.Errorf("ReadAll() error = %v, want %v", err, errBad)
	}
}

func TestReadAllInvalidLimit(t *testing.T) {
	read := func(ctx context.Context, n int) (int, error) {
		t.Error("read must not be called with an invalid limit")
		return 0, nil
	}

	for _, limit := range []int{0, -1} {
		_, err := sourceread.ReadAll(t.Context(), []int{1}, limit, read)
		if !errors.Is(err, sourceread.ErrInvalidLimit) {
			t.Errorf("ReadAll(limit=%d) error = %v, want %v", limit, err, sourceread.ErrInvalidLimit)
		}
	}
}
