// Package visits 統計頁面的造訪次數，並可以在寫入時留下稽核紀錄。
package visits

import "sync"

// Store 讀寫每個路徑的造訪次數。
type Store interface {
	// Incr 記一次造訪。
	Incr(path string)
	// Count 回傳路徑的造訪次數。
	Count(path string) int
	// Reset 把路徑的次數歸零。
	Reset(path string)
}

// ANCHOR: counter

// Counter 把次數存在記憶體裡，可以並行使用。零值可以直接使用。
type Counter struct {
	sync.Mutex
	hits map[string]int
}

// Incr 記一次造訪。
func (c *Counter) Incr(path string) {
	c.Lock()
	defer c.Unlock()
	if c.hits == nil {
		c.hits = make(map[string]int)
	}
	c.hits[path]++
}

// ANCHOR_END: counter

// Count 回傳路徑的造訪次數。
func (c *Counter) Count(path string) int {
	c.Lock()
	defer c.Unlock()
	return c.hits[path]
}

// Reset 把路徑的次數歸零。
func (c *Counter) Reset(path string) {
	c.Lock()
	defer c.Unlock()
	delete(c.hits, path)
}

// ANCHOR: audited

// Audited 包裝一個 Store，每次 Incr 都呼叫 Log 留下紀錄。
type Audited struct {
	Store
	Log func(msg string)
}

// Incr 先留下稽核紀錄，再轉給 Store。
func (a *Audited) Incr(path string) {
	a.Log("incr " + path)
	a.Store.Incr(path)
}

// ANCHOR_END: audited
