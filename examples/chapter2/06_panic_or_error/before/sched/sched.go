// Package sched 解析 "every 5m" 這種重複執行的排程字串。
package sched

import (
	"fmt"
	"strings"
	"time"
)

// Schedule 是解析後的排程。
type Schedule struct {
	// Interval 是兩次執行之間的間隔。
	Interval time.Duration
}

// ANCHOR: parse

// Parse 解析 "every <duration>" 格式的排程字串，例如 "every 5m"。
func Parse(spec string) Schedule {
	rest, ok := strings.CutPrefix(spec, "every ")
	if !ok {
		panic(fmt.Sprintf("parse schedule %q: want \"every <duration>\"", spec))
	}
	d, err := time.ParseDuration(rest)
	if err != nil {
		panic(fmt.Sprintf("parse schedule %q: %v", spec, err))
	}
	if d <= 0 {
		panic(fmt.Sprintf("parse schedule %q: interval must be positive", spec))
	}
	return Schedule{Interval: d}
}

// ANCHOR_END: parse
