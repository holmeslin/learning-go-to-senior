// Package tagset 管理一組標籤：不分大小寫、忽略前後空白。
package tagset

import (
	"slices"
	"strings"
)

// Set 是一組正規化過的標籤，內部維持排好序的 slice。零值不能用，請用 New 建立。
type Set struct {
	items []string
}

// New 回傳一個空的 Set。
func New() *Set {
	return &Set{}
}

// Add 去掉 tag 的前後空白並轉成小寫，再加進去。標籤是新的才回傳 true；
// 空白的標籤會被忽略。
func (s *Set) Add(tag string) bool {
	tag = normalize(tag)
	if tag == "" || s.indexOf(tag) >= 0 {
		return false
	}
	i, _ := slices.BinarySearch(s.items, tag)
	s.items = slices.Insert(s.items, i, tag)
	return true
}

// Contains 回報 tag 去掉前後空白並轉成小寫之後，是否在集合裡。
func (s *Set) Contains(tag string) bool {
	return s.indexOf(normalize(tag)) >= 0
}

// Len 回傳標籤的數量。
func (s *Set) Len() int {
	return len(s.items)
}

// List 依字母順序回傳所有標籤。呼叫端可以修改回傳的 slice。
func (s *Set) List() []string {
	return slices.Clone(s.items)
}

func (s *Set) indexOf(tag string) int {
	if i, found := slices.BinarySearch(s.items, tag); found {
		return i
	}
	return -1
}

func normalize(tag string) string {
	return strings.ToLower(strings.TrimSpace(tag))
}
