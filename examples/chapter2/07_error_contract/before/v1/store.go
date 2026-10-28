// Package v1 是 store 套件的 v1：找不到時回傳哨兵錯誤 ErrNotFound。
package v1

import "errors"

var users = map[int]string{1: "amy", 2: "bob"}

// ErrNotFound 表示沒有這個 id 的使用者。
var ErrNotFound = errors.New("user not found")

// ANCHOR: find

// Find 回傳 id 對應的使用者名稱。找不到時回傳 ErrNotFound。
func Find(id int) (string, error) {
	name, ok := users[id]
	if !ok {
		return "", ErrNotFound
	}
	return name, nil
}

// ANCHOR_END: find
