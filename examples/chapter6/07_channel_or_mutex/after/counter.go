// Package hits 用 mutex 保護各路徑的點閱數。
package hits

import "sync"

// ANCHOR: counter

// Counter 記錄各路徑的點閱數，可以同時被多個 goroutine 使用。
// 零值就能用；使用後不可複製。
type Counter struct {
	mu     sync.Mutex
	counts map[string]int
}

// Inc 把 path 的點閱數加一。
func (c *Counter) Inc(path string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.counts == nil {
		c.counts = make(map[string]int)
	}
	c.counts[path]++
}

// Count 回傳 path 目前的點閱數。
func (c *Counter) Count(path string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.counts[path]
}

// ANCHOR_END: counter
