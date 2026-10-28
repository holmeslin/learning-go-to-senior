// Package route 把請求路徑整理成可以當指標標籤的樣板，避免每個使用者 ID 都變成一個新標籤。
package route

import "strings"

// Template 把 path 裡全由數字組成的路徑段換成 ":id"，其他段維持原樣。
// 結果只配置一次：先量好需要的長度，再一次寫完。
func Template(path string) string {
	var b strings.Builder
	// 每個數字段最多從 1 個字元變成 3 個字元，所以每段多留 2 個位元組就夠。
	b.Grow(len(path) + 2*(strings.Count(path, "/")+1))
	for rest := path; ; {
		seg, next, more := strings.Cut(rest, "/")
		if isID(seg) {
			b.WriteString(":id")
		} else {
			b.WriteString(seg)
		}
		if !more {
			return b.String()
		}
		b.WriteByte('/')
		rest = next
	}
}

// isID 回報 seg 是否非空且全由數字組成。
func isID(seg string) bool {
	if seg == "" {
		return false
	}
	for i := 0; i < len(seg); i++ {
		if seg[i] < '0' || seg[i] > '9' {
			return false
		}
	}
	return true
}
