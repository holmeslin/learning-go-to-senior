// Package slugtool 把文章標題轉成可以放進網址的 slug。這是 v2 的 API。
package slugtool

import (
	"errors"
	"strings"
)

// Slug 把 title 轉成小寫，並用連字號串起以空白分隔的字。
// title 沒有任何可用的字元時回傳錯誤，不再回傳空字串。
func Slug(title string) (string, error) {
	words := strings.Fields(title)
	if len(words) == 0 {
		return "", errors.New("slugtool: title has no usable characters")
	}
	return strings.ToLower(strings.Join(words, "-")), nil
}
