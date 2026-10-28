// Package rle 用連續長度編碼（run-length encoding）壓縮字串：把連續相同的位元組寫成「長度、冒號、位元組」。
package rle

import (
	"fmt"
	"strconv"
	"strings"
)

// maxRun 是 Decode 接受的單一連續長度上限，避免惡意輸入要求配置巨大的記憶體。
const maxRun = 1 << 20

// ANCHOR: encode

// Encode 把 s 壓縮成「長度、冒號、位元組」的序列，例如 "aaab" 變成 "3:a1:b"。
// 冒號標出長度在哪裡結束，所以 s 裡的數字不會和長度混在一起。
func Encode(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		j := i
		for j < len(s) && s[j] == s[i] {
			j++
		}
		b.WriteString(strconv.Itoa(j - i))
		b.WriteByte(':')
		b.WriteByte(s[i])
		i = j
	}
	return b.String()
}

// ANCHOR_END: encode

// Decode 還原 Encode 的結果。格式不對時回傳錯誤。
func Decode(s string) (string, error) {
	var b strings.Builder
	for i := 0; i < len(s); {
		j := i
		for j < len(s) && s[j] >= '0' && s[j] <= '9' {
			j++
		}
		if j == i || j+1 >= len(s) || s[j] != ':' {
			return "", fmt.Errorf("decode: want count, colon and a byte at offset %d", i)
		}
		n, err := strconv.Atoi(s[i:j])
		if err != nil || n > maxRun {
			return "", fmt.Errorf("decode: count %q is too large", s[i:j])
		}
		b.WriteString(strings.Repeat(s[j+1:j+2], n))
		i = j + 2
	}
	return b.String(), nil
}
