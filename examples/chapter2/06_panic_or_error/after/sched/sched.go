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
// 格式不對或間隔不是正數時回傳 error。
func Parse(spec string) (Schedule, error) {
	rest, ok := strings.CutPrefix(spec, "every ")
	if !ok {
		return Schedule{}, fmt.Errorf("parse schedule %q: want \"every <duration>\"", spec)
	}
	d, err := time.ParseDuration(rest)
	if err != nil {
		return Schedule{}, fmt.Errorf("parse schedule %q: %v", spec, err)
	}
	if d <= 0 {
		return Schedule{}, fmt.Errorf("parse schedule %q: interval must be positive", spec)
	}
	return Schedule{Interval: d}, nil
}

// ANCHOR_END: parse

// ANCHOR: must

// MustParse 和 Parse 一樣，但格式不對時會 panic，panic 的值就是 Parse 回傳的 error。
// 只用在寫死在程式裡的字串，例如初始化套件層級的變數；來自使用者或設定檔的字串請用 Parse。
func MustParse(spec string) Schedule {
	s, err := Parse(spec)
	if err != nil {
		panic(err)
	}
	return s
}

// ANCHOR_END: must
