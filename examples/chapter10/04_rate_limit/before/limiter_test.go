package ratelimit_test

import (
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	ratelimit "github.com/holmeslin/learning-go-to-senior/examples/chapter10/04_rate_limit/before"
)

// ANCHOR: problems

func TestSleepLimiterLetsConcurrentCallersThrough(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		lim := ratelimit.NewSleepLimiter(5) // 每秒 5 個
		start := time.Now()

		var inFirstSecond atomic.Int32
		var wg sync.WaitGroup
		for range 20 {
			wg.Go(func() {
				lim.Wait()
				if time.Since(start) < time.Second {
					inFirstSecond.Add(1)
				}
			})
		}
		wg.Wait()

		// 20 個呼叫同時睡 200 毫秒、同時醒來：第一秒內過了 20 個，不是 5 個。
		if got := inFirstSecond.Load(); got != 20 {
			t.Errorf("%d calls got through in the first second, want 20 (the limit says 5)", got)
		}
	})
}

func TestSleepLimiterHasNoBurst(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		lim := ratelimit.NewSleepLimiter(5)
		start := time.Now()

		lim.Wait() // 閒置了很久之後的第一個呼叫

		if got := time.Since(start); got != 200*time.Millisecond {
			t.Errorf("an idle caller waited %v, want 200ms", got)
		}
	})
}

func TestWindowCounterDoublesAcrossTheBoundary(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		lim := ratelimit.NewWindowCounter(5) // 每秒最多 5 個
		drain := func() (accepted int) {
			for lim.Allow() {
				accepted++
			}
			return accepted
		}

		time.Sleep(900 * time.Millisecond) // bubble 的時鐘從整數秒開始，現在是 0.9 秒
		accepted := drain()
		time.Sleep(100 * time.Millisecond) // 1.0 秒：換了一個視窗，計數歸零
		accepted += drain()

		// 100 毫秒內放行了 10 個，是上限 5 的兩倍。
		if accepted != 10 {
			t.Errorf("%d calls accepted within 100ms, want 10 (the limit says 5 per second)", accepted)
		}
	})
}

// ANCHOR_END: problems
