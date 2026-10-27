// Package v11 是 store 套件的 v1.1：錯誤訊息改成帶上 id，其他都沒動。
package v11

import (
	"errors"
	"fmt"
)

var users = map[int]string{1: "amy", 2: "bob"}

// ErrNotFound 表示沒有這個 id 的使用者。
var ErrNotFound = errors.New("user not found")

// ANCHOR: find

// Find 回傳 id 對應的使用者名稱。
func Find(id int) (string, error) {
	name, ok := users[id]
	if !ok {
		return "", fmt.Errorf("user %d not found", id)
	}
	return name, nil
}

// ANCHOR_END: find
