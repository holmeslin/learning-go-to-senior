package fetch_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	fetch "github.com/holmeslin/learning-go-to-senior/examples/chapter6/02_errgroup/after"
)

// step 描述一個假來源：等 delay 之後回傳 err；等的時候 ctx 取消就提早結束。
type step struct {
	delay time.Duration
	err   error
}

// newGet 依 steps 做出一個 fetch.Fetch，並記錄有幾個來源是被取消而結束的。
func newGet(steps map[string]step, canceled *atomic.Int32) fetch.Fetch {
	return func(ctx context.Context, src string) error {
		s := steps[src]
		select {
		case <-time.After(s.delay):
			return s.err
		case <-ctx.Done():
			canceled.Add(1)
			return ctx.Err()
		}
	}
}

// ANCHOR: firsterror

func TestFetchAllReturnsFirstError(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		errA, errB := errors.New("a failed"), errors.New("b failed")
		var canceled atomic.Int32
		get := newGet(map[string]step{
			"a": {10 * time.Millisecond, errA},
			"b": {20 * time.Millisecond, errB},
			"c": {time.Hour, nil},
		}, &canceled)

		start := time.Now()
		err := fetch.FetchAll(t.Context(), []string{"a", "b", "c"}, get)

		if !errors.Is(err, errA) {
			t.Errorf("FetchAll() = %v, want %v", err, errA)
		}
		if got, want := time.Since(start), 10*time.Millisecond; got != want {
			t.Errorf("FetchAll() took %v, want %v", got, want)
		}
		if got, want := canceled.Load(), int32(2); got != want {
			t.Errorf("canceled sources = %d, want %d", got, want)
		}
	})
}

// ANCHOR_END: firsterror

func TestFetchAllSuccess(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var canceled atomic.Int32
		get := newGet(map[string]step{
			"a": {10 * time.Millisecond, nil},
			"b": {20 * time.Millisecond, nil},
		}, &canceled)

		start := time.Now()
		err := fetch.FetchAll(t.Context(), []string{"a", "b"}, get)

		if err != nil {
			t.Errorf("FetchAll() = %v, want nil", err)
		}
		if got, want := time.Since(start), 20*time.Millisecond; got != want {
			t.Errorf("FetchAll() took %v, want %v", got, want)
		}
		if got := canceled.Load(); got != 0 {
			t.Errorf("canceled sources = %d, want 0", got)
		}
	})
}

func TestFetchAllJoinKeepsEveryError(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		errA, errB := errors.New("a failed"), errors.New("b failed")
		var canceled atomic.Int32
		get := newGet(map[string]step{
			"a": {10 * time.Millisecond, errA},
			"b": {20 * time.Millisecond, errB},
		}, &canceled)

		err := fetch.FetchAllJoin(t.Context(), []string{"a", "b"}, get)

		if !errors.Is(err, errA) || !errors.Is(err, errB) {
			t.Errorf("FetchAllJoin() = %v, want both %v and %v", err, errA, errB)
		}
		if got := canceled.Load(); got != 0 {
			t.Errorf("canceled sources = %d, want 0", got)
		}
	})
}
