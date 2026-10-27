// Package text 放處理顯示用文字的函式。
package text

// Truncate 最多保留 s 的前 n 個字元（rune），被截掉時在結尾加上 "…"。
func Truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
