// Package kv 是一個很小的 key-value 儲存：每個值各存成一個檔案。
package kv

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var (
	// ErrNotFound 表示這個 key 底下沒有值。
	ErrNotFound = errors.New("kv: key not found")
	// ErrInvalidKey 表示 key 不合規定：key 就是檔名，所以不能有目錄結構。
	ErrInvalidKey = errors.New("kv: invalid key")
)

// ANCHOR: validkey

// ValidKey 回報 key 能不能用在 Store 上。
// key 必須是單純的檔名：不能是空字串、不能含斜線，也不能是 "." 或 ".."。
func ValidKey(key string) bool {
	return fs.ValidPath(key) && key != "." && !strings.Contains(key, "/")
}

// ANCHOR_END: validkey

// DirStore 把每個值存成一個檔案，檔名就是 key，全部放在同一個目錄裡。
type DirStore struct {
	dir string
}

// NewDirStore 回傳使用 dir 的 DirStore；dir 必須已經存在。
func NewDirStore(dir string) *DirStore {
	return &DirStore{dir: dir}
}

// Put 把 value 存在 key 底下，取代原本的值。
// key 不合規定時回傳 ErrInvalidKey。
func (s *DirStore) Put(key, value string) error {
	if !ValidKey(key) {
		return ErrInvalidKey
	}
	if err := os.WriteFile(filepath.Join(s.dir, key), []byte(value), 0o600); err != nil {
		return fmt.Errorf("kv: put: %w", err)
	}
	return nil
}

// Get 回傳 key 底下的值。
// key 不合規定時回傳 ErrInvalidKey，沒有值時回傳 ErrNotFound。
func (s *DirStore) Get(key string) (string, error) {
	if !ValidKey(key) {
		return "", ErrInvalidKey
	}
	b, err := os.ReadFile(filepath.Join(s.dir, key))
	if errors.Is(err, fs.ErrNotExist) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("kv: get: %w", err)
	}
	return string(b), nil
}
