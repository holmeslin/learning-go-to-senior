package accesslog

import (
	"hash/maphash"
	"sync"
)

// ANCHOR: metrics

const shardCount = 16

// seed 讓同一個 key 在整個程式裡永遠落在同一個分片。
var seed = maphash.MakeSeed()

// Metrics 統計各「方法 路徑」的請求數，可以同時被多個 goroutine 使用。
// key 依雜湊值分到 16 個分片，每個分片各有一把鎖，不同分片的請求不必互相等。
// 零值就能用；使用後不可複製。
type Metrics struct {
	shards [shardCount]struct {
		mu     sync.Mutex
		counts map[string]int
	}
}

// Observe 記錄一筆請求。
func (m *Metrics) Observe(method, path string) {
	key := method + " " + path // 鎖外面先準備好 key，臨界區只剩 map 的更新
	s := &m.shards[maphash.String(seed, key)%shardCount]

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.counts == nil {
		s.counts = make(map[string]int)
	}
	s.counts[key]++
}

// ANCHOR_END: metrics

// Count 回傳 key（格式為「方法 路徑」）目前的請求數。
func (m *Metrics) Count(key string) int {
	s := &m.shards[maphash.String(seed, key)%shardCount]

	s.mu.Lock()
	defer s.mu.Unlock()
	return s.counts[key]
}
