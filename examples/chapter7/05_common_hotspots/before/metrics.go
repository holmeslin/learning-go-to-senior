package accesslog

import "sync"

// ANCHOR: metrics

// Metrics 統計各「方法 路徑」的請求數，可以同時被多個 goroutine 使用。
// 零值就能用；使用後不可複製。
type Metrics struct {
	mu     sync.Mutex
	counts map[string]int
}

// Observe 記錄一筆請求。
func (m *Metrics) Observe(method, path string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.counts == nil {
		m.counts = make(map[string]int)
	}
	key := method + " " + path
	m.counts[key]++
}

// ANCHOR_END: metrics

// Count 回傳 key（格式為「方法 路徑」）目前的請求數。
func (m *Metrics) Count(key string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.counts[key]
}
