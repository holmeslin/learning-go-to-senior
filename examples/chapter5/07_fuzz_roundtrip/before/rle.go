// Package rle 用連續長度編碼（run-length encoding）壓縮字串：把連續相同的位元組寫成「長度加位元組」。
package rle

import (
	"fmt"
	"strconv"
	"strings"
)

// maxRun 是 Decode 接受的單一連續長度上限，避免惡意輸入要求配置巨大的記憶體。
const maxRun = 1 << 20

// ANCHOR: encode

// Encode 把 s 壓縮成「長度加位元組」的序列，例如 "aaab" 變成 "3a1b"。
func Encode(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		j := i
		for j < len(s) && s[j] == s[i] {
			j++
		}
		b.WriteString(strconv.Itoa(j - i))
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
		if j == i {
			return "", fmt.Errorf("decode: no count at offset %d", i)
		}
		if j == len(s) {
			return "", fmt.Errorf("decode: count %q has no byte after it", s[i:j])
		}
		n, err := strconv.Atoi(s[i:j])
		if err != nil || n > maxRun {
			return "", fmt.Errorf("decode: count %q is too large", s[i:j])
		}
		b.WriteString(strings.Repeat(s[j:j+1], n))
		i = j + 1
	}
	return b.String(), nil
}
