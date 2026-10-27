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

// ANCHOR: new

// New 建立一個 Cache。maxValueLen 為 0 表示不限制值的長度，onEvict 可以是 nil。
func New(capacity, maxValueLen int, onEvict func(key string)) *Cache {
	return &Cache{
		capacity:    capacity,
		maxValueLen: maxValueLen,
		onEvict:     onEvict,
		items:       make(map[string]string),
	}
}

// ANCHOR_END: new

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
