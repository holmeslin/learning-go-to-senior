// Package cache 提供固定容量的快取，超過容量時淘汰最久沒用的項目。
package cache

import "github.com/holmeslin/learning-go-to-senior/examples/chapter1/03_internal_or_public/after/cache/internal/lru"

// Stats 是快取的命中統計。
type Stats struct {
	Hits, Misses int
}

// ANCHOR: cache
// Cache 是固定容量的快取。淘汰順序怎麼記錄，是 internal/lru 的事。
type Cache[K comparable, V any] struct {
	order *lru.List[K, V]
	stats Stats
}

// New 建立容量為 capacity 的 Cache；capacity 小於 1 時當作 1。
func New[K comparable, V any](capacity int) *Cache[K, V] {
	return &Cache[K, V]{order: lru.New[K, V](max(capacity, 1))}
}

// ANCHOR_END: cache

// Get 回傳 key 對應的值，並記錄命中或未命中。
func (c *Cache[K, V]) Get(key K) (V, bool) {
	val, ok := c.order.Get(key)
	if ok {
		c.stats.Hits++
	} else {
		c.stats.Misses++
	}
	return val, ok
}

// Put 寫入 key；超過容量時淘汰最久沒用的項目。
func (c *Cache[K, V]) Put(key K, val V) {
	c.order.Put(key, val)
}

// Stats 回傳目前的命中統計。
func (c *Cache[K, V]) Stats() Stats {
	return c.stats
}
