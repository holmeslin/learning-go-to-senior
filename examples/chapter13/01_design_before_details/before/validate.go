package signup

import "strings"

// normalize 去掉名稱前後的空白；結果是空的回傳 ErrEmptyName。
func normalize(who string) (string, error) {
	who = strings.TrimSpace(who)
	if who == "" {
		return "", ErrEmptyName
	}
	return who, nil
}
