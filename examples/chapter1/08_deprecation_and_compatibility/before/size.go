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
		return 0, fmt.Errorf("size: %q has no unit", s)
	}
	n, err := strconv.ParseInt(s[:i], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("size: %q has an invalid number: %w", s, err)
	}
	unit, ok := units[strings.TrimSpace(s[i:])]
	if !ok {
		return 0, fmt.Errorf("size: %q has an unknown unit", s)
	}
	if n > math.MaxInt64/unit {
		return 0, fmt.Errorf("size: %q is too large", s)
	}
	return n * unit, nil
}

// ANCHOR_END: parse
