// Package ingest 是 webhook 接收服務裡三個元件修正後的版本：快取有筆數上限、
// 保留的請求代號是自己的副本、處理中的通知表可以重建。
package ingest

import "container/list"

// ANCHOR: cache

// Cache 記住每個客戶最近一次算好的摘要，筆數有上限，滿了就丟掉最久沒用到的。
type Cache struct {
	max   int
	order *list.List // 最近用過的在前面
	items map[string]*list.Element
}

type entry struct{ key, value string }

// NewCache 回傳最多記住 limit 筆的 Cache；limit 小於 1 時當作 1。
func NewCache(limit int) *Cache {
	return &Cache{max: max(limit, 1), order: list.New(), items: make(map[string]*list.Element)}
}

// Get 回傳客戶的摘要並標成最近用過；沒有記錄時第二個回傳值是 false。
func (c *Cache) Get(key string) (string, bool) {
	el, ok := c.items[key]
	if !ok {
		return "", false
	}
	c.order.MoveToFront(el)
	return el.Value.(entry).value, true
}

// Put 記下客戶的摘要，已有的值會被覆蓋；筆數超過上限時丟掉最久沒用到的。
func (c *Cache) Put(key, value string) {
	if el, ok := c.items[key]; ok {
		el.Value = entry{key, value}
		c.order.MoveToFront(el)
		return
	}
	c.items[key] = c.order.PushFront(entry{key, value})
	if c.order.Len() > c.max {
		oldest := c.order.Remove(c.order.Back()).(entry)
		delete(c.items, oldest.key)
	}
}

// Len 回傳目前記住的客戶數。
func (c *Cache) Len() int {
	return c.order.Len()
}

// ANCHOR_END: cache
