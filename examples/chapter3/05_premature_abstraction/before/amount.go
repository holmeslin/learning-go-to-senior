// Package amount 把金額格式化成字串。這是早早抽出共用函式的版本：
// 每多一個呼叫端，Format 就多一個旗標。
package amount

import (
	"fmt"
	"strconv"
)

// ANCHOR: format

// Format 把 cents（單位是分）格式化成金額字串。
// symbol 加上 $；thousands 在整數部分加千分位；parens 用括號表示負數（會計格式）。
func Format(cents int64, symbol, thousands, parens bool) string {
	neg := cents < 0
	n := uint64(cents)
	if neg {
		n = -n
	}
	whole := strconv.FormatUint(n/100, 10)
	if thousands {
		whole = group(whole)
	}
	s := fmt.Sprintf("%s.%02d", whole, n%100)
	if parens && cents == 0 {
		s = "-" // 會計格式：零顯示成 -
	}
	if symbol {
		s = "$" + s
	}
	if neg {
		if parens {
			s = "(" + s + ")"
		} else {
			s = "-" + s
		}
	}
	return s
}

// ANCHOR_END: format

// group 在整數部分每三位加一個逗號。
func group(digits string) string {
	for i := len(digits) - 3; i > 0; i -= 3 {
		digits = digits[:i] + "," + digits[i:]
	}
	return digits
}
