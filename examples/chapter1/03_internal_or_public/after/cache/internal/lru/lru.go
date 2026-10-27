// Package lru 是 cache 用來決定淘汰順序的實作細節，不給 cache 以外的套件使用。
package lru

import "container/list"

type entry[K comparable, V any] struct {
	key K
	val V
}

// List 保存最多 capacity 個項目，最久沒用的排在最後面。
type List[K comparable, V any] struct {
	capacity int
	order    *list.List
	items    map[K]*list.Element
}

// New 建立一個最多保存 capacity 個項目的 List。
func New[K comparable, V any](capacity int) *List[K, V] {
	return &List[K, V]{
		capacity: capacity,
		order:    list.New(),
		items:    make(map[K]*list.Element),
	}
}

// Get 回傳 key 對應的值，並把它標記為剛用過。
func (l *List[K, V]) Get(key K) (V, bool) {
	el, ok := l.items[key]
	if !ok {
		var zero V
		return zero, false
	}
	l.order.MoveToFront(el)
	return el.Value.(*entry[K, V]).val, true
}

// Put 寫入 key；超過容量時淘汰最久沒用的項目。
func (l *List[K, V]) Put(key K, val V) {
	if el, ok := l.items[key]; ok {
		el.Value.(*entry[K, V]).val = val
		l.order.MoveToFront(el)
		return
	}
	l.items[key] = l.order.PushFront(&entry[K, V]{key, val})
	if l.order.Len() > l.capacity {
		oldest := l.order.Back()
		l.order.Remove(oldest)
		delete(l.items, oldest.Value.(*entry[K, V]).key)
	}
}
