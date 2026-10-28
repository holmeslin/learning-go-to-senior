package hits_test

import (
	"sync"
	"testing"

	hits "github.com/holmeslin/learning-go-to-senior/examples/chapter6/07_channel_or_mutex/before"
)

func TestCountsAfterInc(t *testing.T) {
	c := hits.New()
	defer c.Close()

	if got := c.Count("/home"); got != 0 {
		t.Errorf("Count(/home) before any Inc = %d, want 0", got)
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

func TestConcurrentIncAndCount(t *testing.T) {
	const workers, perWorker = 8, 500
	c := hits.New()
	defer c.Close()

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
