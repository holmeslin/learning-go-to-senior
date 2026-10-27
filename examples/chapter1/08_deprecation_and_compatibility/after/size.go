// Package size 解析 "10MiB" 這類人類可讀的大小字串。
package size

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

var units = map[string]int64{"B": 1, "KiB": 1 << 10, "MiB": 1 << 20, "GiB": 1 << 30}

// ANCHOR: parse

// Parse 把 "10MiB" 這樣的字串解析成位元組數。支援 B、KiB、MiB、GiB。
func Parse(s string) (int64, error) {
	s = strings.TrimSpace(s)
	i := strings.IndexFunc(s, func(r rune) bool { return r < '0' || r > '9' })
	if i < 0 {
		return 0, fmt.Errorf("size: %q 沒有單位", s)
	}
	n, err := strconv.ParseInt(s[:i], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("size: %q 的數字不正確: %w", s, err)
	}
	unit, ok := units[strings.TrimSpace(s[i:])]
	if !ok {
		return 0, fmt.Errorf("size: %q 的單位不認得", s)
	}
	if n > math.MaxInt64/unit {
		return 0, fmt.Errorf("size: %q 太大", s)
	}
	return n * unit, nil
}

// ANCHOR_END: parse

// ANCHOR: deprecated

// ParseSize 把大小字串解析成位元組數。
//
// Deprecated: 套件名稱已經是 size，名稱重複了；請改用 [Parse]。
//
//go:fix inline
func ParseSize(s string) (int64, error) {
	return Parse(s)
}

// ANCHOR_END: deprecated
