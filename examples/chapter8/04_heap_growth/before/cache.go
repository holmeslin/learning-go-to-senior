// Package ingest 是 webhook 接收服務裡三個會讓 heap 一路成長的小元件。
package ingest

// ANCHOR: cache

// Cache 記住每個客戶最近一次算好的摘要。
type Cache struct {
	m map[string]string
}

// NewCache 回傳空的 Cache。
func NewCache() *Cache {
	return &Cache{m: make(map[string]string)}
}

// Get 回傳客戶的摘要；沒有記錄時第二個回傳值是 false。
func (c *Cache) Get(key string) (string, bool) {
	v, ok := c.m[key]
	return v, ok
}

// Put 記下客戶的摘要，已有的值會被覆蓋。
func (c *Cache) Put(key, value string) {
	c.m[key] = value
}

// Len 回傳目前記住的客戶數。
func (c *Cache) Len() int {
	return len(c.m)
}

// ANCHOR_END: cache
