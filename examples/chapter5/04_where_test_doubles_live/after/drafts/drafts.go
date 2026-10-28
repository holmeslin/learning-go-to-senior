// Package drafts autosaves text being edited and restores it later.
package drafts

import (
	"errors"
	"fmt"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter5/04_where_test_doubles_live/after/kv"
)

// Store is the part of a key-value store that drafts needs.
type Store interface {
	Put(key, value string) error
	Get(key string) (string, error)
}

// Autosave saves text as the draft called title.
func Autosave(s Store, title, text string) error {
	if err := s.Put(title, text); err != nil {
		return fmt.Errorf("autosave %q: %w", title, err)
	}
	return nil
}

// Restore returns the draft called title, or "" if there is none.
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
