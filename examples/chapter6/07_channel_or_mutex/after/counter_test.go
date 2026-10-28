package hits_test

import (
	"sync"
	"testing"

	hits "github.com/holmeslin/learning-go-to-senior/examples/chapter6/07_channel_or_mutex/after"
)

func TestZeroValueCounts(t *testing.T) {
	var c hits.Counter
	if got := c.Count("/home"); got != 0 {
		t.Errorf("Count(/home) on zero value = %d, want 0", got)
	}

	c.Inc("/home")
	c.Inc("/home")
	c.Inc("/about")

	if got, want := c.Count("/home"), 2; got != want {
		t.Errorf("Count(/home) = %d, want %d", got, want)
	}
	if got, want := c.Count("/about"), 1; got != want {
		t.Errorf("Count(/about) = %d, want %d", got, want)
	}
}

// ANCHOR: concurrent

// 多個 goroutine 同時寫入與讀取；少了同步，這個測試在 -race 下會失敗。
func TestConcurrentIncAndCount(t *testing.T) {
	const workers, perWorker = 8, 500
	var c hits.Counter

	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() {
			for range perWorker {
				c.Inc("/home")
				_ = c.Count("/home") // 讀寫交錯
			}
		})
	}
	wg.Wait()

	if got, want := c.Count("/home"), workers*perWorker; got != want {
		t.Errorf("Count(/home) = %d, want %d", got, want)
	}
}

// ANCHOR_END: concurrent
