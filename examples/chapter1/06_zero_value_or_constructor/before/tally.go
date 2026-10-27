// Package tally 統計頁面造訪次數並輸出報表。
package tally

import (
	"fmt"
	"io"
	"maps"
	"slices"
)

// ANCHOR: counter

// Counter 統計每個路徑的造訪次數。
type Counter struct {
	hits map[string]int
}

// NewCounter 建立一個可以使用的 Counter。
func NewCounter() *Counter {
	return &Counter{hits: make(map[string]int)}
}

// Add 記一次造訪。
func (c *Counter) Add(path string) {
	c.hits[path]++
}

// ANCHOR_END: counter

// Count 回傳路徑的造訪次數。
func (c *Counter) Count(path string) int {
	return c.hits[path]
}

// Paths 回傳造訪過的路徑，由小到大排序。
func (c *Counter) Paths() []string {
	return slices.Sorted(maps.Keys(c.hits))
}

// ANCHOR: reporter

// Reporter 把 Counter 的內容寫到 io.Writer。
type Reporter struct {
	w io.Writer
}

// NewReporter 建立寫到 w 的 Reporter。
func NewReporter(w io.Writer) *Reporter {
	return &Reporter{w: w}
}

// ANCHOR_END: reporter

// Report 每個路徑輸出一行「路徑 次數」。
func (r *Reporter) Report(c *Counter) {
	for _, p := range c.Paths() {
		fmt.Fprintf(r.w, "%s %d\n", p, c.Count(p))
	}
}
