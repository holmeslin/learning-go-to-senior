// Package store 提供使用者查詢。
//
// 錯誤契約：Find 回傳的錯誤見它的文件註解。呼叫端請用 errors.Is 與
// errors.As 判斷，不要比對錯誤訊息；訊息可能隨版本調整，不屬於契約。
package store

import (
	"errors"
	"fmt"
)

var users = map[int]string{1: "amy", 2: "bob"}

// ErrNotFound 表示沒有這個 id 的使用者。
var ErrNotFound = errors.New("user not found")

// InvalidIDError 表示 id 不合法（小於 1）。
type InvalidIDError struct {
	ID int // 呼叫端傳入的 id
}

func (e *InvalidIDError) Error() string {
	return fmt.Sprintf("invalid user id %d", e.ID)
}

// ANCHOR: find

// Find 回傳 id 對應的使用者名稱。
//
// 回傳的錯誤屬於 API 契約：
//   - id 小於 1：回傳 *InvalidIDError，用 errors.AsType 取出。
//   - 找不到使用者：回傳的錯誤符合 errors.Is(err, ErrNotFound)；
//     錯誤可能帶有額外的上下文，不要用 == 比較。
//
// 除此之外不會回傳其他錯誤。
func Find(id int) (string, error) {
	if id < 1 {
		return "", &InvalidIDError{ID: id}
	}
	name, ok := users[id]
	if !ok {
		return "", fmt.Errorf("find user %d: %w", id, ErrNotFound)
	}
	return name, nil
}

// ANCHOR_END: find
