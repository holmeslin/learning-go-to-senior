// Package reports 讓使用者下載報表目錄裡的檔案，檔名來自使用者輸入。
//
// 這個套件是刻意示範的漏洞：路徑檢查可以被繞過，不要照抄到正式程式。
// 可以參考的寫法在同一集的 after 目錄。
package reports

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// ErrOutside 表示檔名指到報表目錄之外。
var ErrOutside = errors.New("path outside reports directory")

// Store 是一個報表目錄。
type Store struct {
	// Dir 是報表目錄的路徑。
	Dir string
}

// ANCHOR: read

// Read 讀出報表目錄裡名為 name 的檔案；name 來自使用者輸入。
func (s *Store) Read(name string) ([]byte, error) {
	full := filepath.Join(s.Dir, name)
	if !strings.HasPrefix(full, s.Dir) {
		return nil, ErrOutside
	}
	return os.ReadFile(full)
}

// ANCHOR_END: read
