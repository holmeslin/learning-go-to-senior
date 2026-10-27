// Package tally 統計頁面造訪次數並輸出報表。
package tally

import (
	"fmt"
	"io"
	"maps"
	"slices"
)

// ANCHOR: counter

// Counter 統計每個路徑的造訪次數。零值可以直接使用。
type Counter struct {
	hits map[string]int
}

// Add 記一次造訪。
func (c *Counter) Add(path string) {
	if c.hits == nil {
		c.hits = make(map[string]int)
	}
	c.hits[path]++
}

// ANCHOR_END: counter

// Count 回傳路徑的造訪次數；讀取 nil map 是安全的。
func (c *Counter) Count(path string) int {
	return c.hits[path]
}

// Paths 回傳造訪過的路徑，由小到大排序。
func (c *Counter) Paths() []string {
	return slices.Sorted(maps.Keys(c.hits))
}

// ANCHOR: reporter

// reporter 把 Counter 的內容寫到 io.Writer。沒有寫入目標就沒有意義，
// 所以型別不匯出，呼叫端只能透過 NewReporter 取得。
type reporter struct {
	w io.Writer
}

// NewReporter 建立寫到 w 的 reporter；w 不可為 nil。
func NewReporter(w io.Writer) *reporter {
	return &reporter{w: w}
}

// ANCHOR_END: reporter

// Report 每個路徑輸出一行「路徑 次數」。
func (r *reporter) Report(c *Counter) {
	for _, p := range c.Paths() {
		fmt.Fprintf(r.w, "%s %d\n", p, c.Count(p))
	}
}
