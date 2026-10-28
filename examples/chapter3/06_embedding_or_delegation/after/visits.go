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
	mu   sync.Mutex
	hits map[string]int
}

// Incr 記一次造訪。
func (c *Counter) Incr(path string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.hits == nil {
		c.hits = make(map[string]int)
	}
	c.hits[path]++
}

// ANCHOR_END: counter

// Count 回傳路徑的造訪次數。
func (c *Counter) Count(path string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.hits[path]
}

// Reset 把路徑的次數歸零。
func (c *Counter) Reset(path string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.hits, path)
}

// ANCHOR: audited

// Audited 包裝一個 Store，每個會寫入的方法都先呼叫 log 留下紀錄。
type Audited struct {
	store Store
	log   func(msg string)
}

// NewAudited 建立包裝 store 的 Audited。store 與 log 都不可以是 nil。
func NewAudited(store Store, log func(msg string)) *Audited {
	return &Audited{store: store, log: log}
}

// Incr 先留下稽核紀錄，再轉給 store。
func (a *Audited) Incr(path string) {
	a.log("incr " + path)
	a.store.Incr(path)
}

// Count 轉給 store；讀取不需要稽核。
func (a *Audited) Count(path string) int { return a.store.Count(path) }

// Reset 先留下稽核紀錄，再轉給 store。
func (a *Audited) Reset(path string) {
	a.log("reset " + path)
	a.store.Reset(path)
}

// ANCHOR_END: audited
