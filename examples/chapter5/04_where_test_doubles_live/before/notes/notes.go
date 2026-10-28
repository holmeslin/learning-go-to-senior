// Package notes 把純文字筆記存在 key-value 儲存裡。
package notes

import (
	"errors"
	"fmt"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter5/04_where_test_doubles_live/before/kv"
)

// Store 是 notes 需要的那一小部分 key-value 儲存。
type Store interface {
	Put(key, value string) error
	Get(key string) (string, error)
}

// Append 把 line 接在名為 title 的筆記後面；筆記不存在就建立它。
func Append(s Store, title, line string) error {
	body, err := s.Get(title)
	if err != nil && !errors.Is(err, kv.ErrNotFound) {
		return fmt.Errorf("append to %q: %w", title, err)
	}
	if body != "" {
		body += "\n"
	}
	if err := s.Put(title, body+line); err != nil {
		return fmt.Errorf("append to %q: %w", title, err)
	}
	return nil
}
