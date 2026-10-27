// Package profile 讀取使用者的個人設定。
package profile

import (
	"fmt"
	"io/fs"
)

// ANCHOR: load

// Load 從 fsys 讀出使用者 name 的設定檔，回傳檔案內容。
func Load(fsys fs.FS, name string) (string, error) {
	data, err := fs.ReadFile(fsys, name+".json")
	if err != nil {
		return "", fmt.Errorf("load profile %q: %w", name, err)
	}
	return string(data), nil
}

// ANCHOR_END: load
