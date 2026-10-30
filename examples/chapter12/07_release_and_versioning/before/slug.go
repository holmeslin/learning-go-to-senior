// Package slugtool 把文章標題轉成可以放進網址的 slug。這是 v1 的 API。
package slugtool

import "strings"

// Slug 把 title 轉成小寫，並用連字號串起以空白分隔的字。
func Slug(title string) string {
	return strings.ToLower(strings.Join(strings.Fields(title), "-"))
}
