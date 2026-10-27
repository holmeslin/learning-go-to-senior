// Package cacheconf 是 cache 的另一種建構方式：用 Config struct 取代 functional options。
package cacheconf

// Cache 是放在記憶體裡的快取。
type Cache struct {
	cfg   Config
	order []string
	items map[string]string
}

// ANCHOR: config

// Config 是 Cache 的設定，每個欄位的零值都有意義。
type Config struct {
	Capacity    int              // 最多保留幾筆；0 表示預設值 100
	MaxValueLen int              // 值的長度上限；0 表示不限制
	OnEvict     func(key string) // 淘汰一筆資料時的回呼；nil 表示不呼叫
}

// New 依 cfg 建立一個 Cache。
func New(cfg Config) *Cache {
	if cfg.Capacity <= 0 {
		cfg.Capacity = 100
	}
	return &Cache{cfg: cfg, items: make(map[string]string)}
}

// ANCHOR_END: config

// Set 寫入一筆資料。值超過長度上限時回傳 false。
func (c *Cache) Set(key, value string) bool {
	if c.cfg.MaxValueLen > 0 && len(value) > c.cfg.MaxValueLen {
		return false
	}
	if _, ok := c.items[key]; !ok {
		if len(c.order) == c.cfg.Capacity {
			oldest := c.order[0]
			c.order = c.order[1:]
			delete(c.items, oldest)
			if c.cfg.OnEvict != nil {
				c.cfg.OnEvict(oldest)
			}
		}
		c.order = append(c.order, key)
	}
	c.items[key] = value
	return true
}

// Get 回傳 key 對應的值。
func (c *Cache) Get(key string) (string, bool) {
	v, ok := c.items[key]
	return v, ok
}
