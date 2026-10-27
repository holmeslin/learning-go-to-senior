// Package profile 讀取使用者的個人設定。
package profile

import (
	"errors"
	"fmt"
	"io/fs"
)

// ErrNotFound 表示該使用者沒有設定檔。用 errors.Is 判斷。
var ErrNotFound = errors.New("profile not found")

// ANCHOR: load

// Load 從 fsys 讀出使用者 name 的設定檔，回傳檔案內容。
// 找不到時，回傳的錯誤符合 ErrNotFound；其他錯誤只保留訊息，不透出底層錯誤。
func Load(fsys fs.FS, name string) (string, error) {
	data, err := fs.ReadFile(fsys, name+".json")
	if errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("load profile %q: %w", name, ErrNotFound)
	}
	if err != nil {
		return "", fmt.Errorf("load profile %q: %v", name, err)
	}
	return string(data), nil
}

// ANCHOR_END: load
