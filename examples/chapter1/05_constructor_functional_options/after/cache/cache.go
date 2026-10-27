// Package cache 是一個固定容量的字串快取，寫入超過容量時淘汰最舊的一筆。
package cache

// Cache 是放在記憶體裡的快取。
type Cache struct {
	capacity    int
	maxValueLen int
	onEvict     func(key string)
	order       []string
	items       map[string]string
}

// ANCHOR: options

// Option 設定 Cache 的一個選項。
type Option func(*Cache)

// WithCapacity 設定最多保留幾筆，預設 100。
func WithCapacity(n int) Option {
	return func(c *Cache) { c.capacity = n }
}

// WithMaxValueLen 設定值的長度上限，超過的值會被 Set 拒絕。預設不限制。
func WithMaxValueLen(n int) Option {
	return func(c *Cache) { c.maxValueLen = n }
}

// WithOnEvict 設定淘汰一筆資料時的回呼。預設不呼叫任何函式。
func WithOnEvict(fn func(key string)) Option {
	return func(c *Cache) { c.onEvict = fn }
}

// New 建立一個 Cache。沒有給的選項使用預設值。
func New(opts ...Option) *Cache {
	c := &Cache{capacity: 100, items: make(map[string]string)}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// ANCHOR_END: options

// Set 寫入一筆資料。值超過長度上限時回傳 false。
func (c *Cache) Set(key, value string) bool {
	if c.maxValueLen > 0 && len(value) > c.maxValueLen {
		return false
	}
	if _, ok := c.items[key]; !ok {
		if len(c.order) == c.capacity {
			oldest := c.order[0]
			c.order = c.order[1:]
			delete(c.items, oldest)
			if c.onEvict != nil {
				c.onEvict(oldest)
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
