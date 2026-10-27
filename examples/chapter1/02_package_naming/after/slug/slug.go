// Package slug 把標題轉成可以放進網址的字串。
package slug

import "strings"

// Make 把 title 轉成小寫，並用 "-" 連接其中的單字。
func Make(title string) string {
	return strings.Join(strings.Fields(strings.ToLower(title)), "-")
}
