// Package util 放各種共用的小函式。
package util

import "strings"

func Slugify(title string) string {
	return strings.Join(strings.Fields(strings.ToLower(title)), "-")
}

func Truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

func Contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
