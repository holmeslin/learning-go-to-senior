// Package config 從檔案讀出伺服器的設定。
package config

import (
	"fmt"
	"io/fs"
	"strconv"
	"strings"
)

// ANCHOR: load

// Load 讀取 name 檔，檔案內容是一個埠號，回傳該埠號。
func Load(fsys fs.FS, name string) (int, error) {
	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		return 0, fmt.Errorf("load config: %w", err)
	}
	port, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("load config: parse port: %w", err)
	}
	return port, nil
}

// ANCHOR_END: load
