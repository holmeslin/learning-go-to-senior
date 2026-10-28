// Package amount 把金額格式化成字串。每個呼叫端有自己的函式，
// 只共用拆數字與千分位這兩個規則一致的部分。
package amount

import (
	"fmt"
	"strconv"
)

// ANCHOR: shared

// split 把 cents（單位是分）拆成正負號、整數部分與兩位小數。
// 絕對值先轉成 uint64 再算，所以 -50 分是負的 0.50，最小的 int64 也不會溢位。
func split(cents int64) (neg bool, whole, frac string) {
	n := uint64(cents)
	if cents < 0 {
		neg, n = true, -n
	}
	return neg, strconv.FormatUint(n/100, 10), fmt.Sprintf("%02d", n%100)
}

// group 在整數部分每三位加一個逗號。
func group(digits string) string {
	for i := len(digits) - 3; i > 0; i -= 3 {
		digits = digits[:i] + "," + digits[i:]
	}
	return digits
}

// ANCHOR_END: shared

// ANCHOR: callers

// CSV 格式化給試算表讀的金額，例如 1234.50；負數用負號，不加符號與千分位。
func CSV(cents int64) string {
	neg, whole, frac := split(cents)
	if neg {
		return "-" + whole + "." + frac
	}
	return whole + "." + frac
}

// Receipt 格式化收據上的金額，例如 $1,234.50；負數（退款）是 -$12.00。
func Receipt(cents int64) string {
	neg, whole, frac := split(cents)
	if neg {
		return "-$" + group(whole) + "." + frac
	}
	return "$" + group(whole) + "." + frac
}

// Ledger 格式化帳務報表的金額，例如 1,234.50；負數用括號，零顯示成 -。
func Ledger(cents int64) string {
	if cents == 0 {
		return "-"
	}
	neg, whole, frac := split(cents)
	if neg {
		return "(" + group(whole) + "." + frac + ")"
	}
	return group(whole) + "." + frac
}

// Invoice 格式化發票上的金額，例如 $1,234.50；負數是 ($12.00)，零是 $0.00。
func Invoice(cents int64) string {
	neg, whole, frac := split(cents)
	if neg {
		return "($" + group(whole) + "." + frac + ")"
	}
	return "$" + group(whole) + "." + frac
}

// ANCHOR_END: callers
