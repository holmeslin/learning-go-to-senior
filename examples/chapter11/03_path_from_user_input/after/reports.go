// Package reports 讓使用者下載報表目錄裡的檔案，檔名來自使用者輸入。
package reports

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ErrInvalidName 表示檔名不合法，在碰到檔案系統之前就被拒絕。
var ErrInvalidName = errors.New("invalid report name")

// Store 是一個報表目錄，所有讀取都被限制在這個目錄裡。
type Store struct {
	root *os.Root
}

// Open 開啟報表目錄 dir。用完要呼叫 [Store.Close]。
func Open(dir string) (*Store, error) {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}
	return &Store{root: root}, nil
}

// Close 關閉報表目錄。
func (s *Store) Close() error {
	return s.root.Close()
}

// ANCHOR: read

// Read 讀出報表目錄裡名為 name 的檔案；name 來自使用者輸入。
// 檔名不合法時回傳 [ErrInvalidName]；符號連結指到目錄外時由 os.Root 拒絕。
func (s *Store) Read(name string) ([]byte, error) {
	if !validName(name) {
		return nil, fmt.Errorf("%w: %q", ErrInvalidName, name)
	}
	return s.root.ReadFile(name)
}

// validName 先在字串上擋掉明顯不對的檔名：IsLocal 擋 ".."、絕對路徑與空字串，
// 副檔名限制則是這個服務自己的規則，只提供 .csv。
func validName(name string) bool {
	return filepath.IsLocal(name) && filepath.Ext(name) == ".csv"
}

// ANCHOR_END: read
