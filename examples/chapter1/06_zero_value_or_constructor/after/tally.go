// Package tally 統計頁面造訪次數並輸出報表。
package tally

import (
	"errors"
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

// ErrNoWriter 表示 Reporter 沒有寫入目標，通常是沒有用 NewReporter 建立。
var ErrNoWriter = errors.New("tally: reporter has no writer, create it with NewReporter")

// Reporter 把 Counter 的內容寫到 io.Writer。請用 NewReporter 建立；
// 直接使用零值時，Report 會回傳 ErrNoWriter。
type Reporter struct {
	w io.Writer
}

// NewReporter 建立寫到 w 的 Reporter。
func NewReporter(w io.Writer) *Reporter {
	return &Reporter{w: w}
}

// Report 每個路徑輸出一行「路徑 次數」。
func (r *Reporter) Report(c *Counter) error {
	if r == nil || r.w == nil {
		return ErrNoWriter
	}
	for _, p := range c.Paths() {
		if _, err := fmt.Fprintf(r.w, "%s %d\n", p, c.Count(p)); err != nil {
			return err
		}
	}
	return nil
}

// ANCHOR_END: reporter
