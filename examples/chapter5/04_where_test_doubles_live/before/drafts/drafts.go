// Package drafts 自動儲存編輯中的文字，之後可以還原。
package drafts

import (
	"errors"
	"fmt"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter5/04_where_test_doubles_live/before/kv"
)

// Store 是 drafts 需要的那一小部分 key-value 儲存。
type Store interface {
	Put(key, value string) error
	Get(key string) (string, error)
}

// Autosave 把 text 存成名為 title 的草稿。
func Autosave(s Store, title, text string) error {
	if err := s.Put(title, text); err != nil {
		return fmt.Errorf("autosave %q: %w", title, err)
	}
	return nil
}

// Restore 回傳名為 title 的草稿；沒有草稿時回傳 ""。
func Restore(s Store, title string) (string, error) {
	text, err := s.Get(title)
	if errors.Is(err, kv.ErrNotFound) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("restore %q: %w", title, err)
	}
	return text, nil
}
