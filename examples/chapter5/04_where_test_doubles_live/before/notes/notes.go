// Package notes keeps plain-text notes in a key-value store.
package notes

import (
	"errors"
	"fmt"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter5/04_where_test_doubles_live/before/kv"
)

// Store is the part of a key-value store that notes needs.
type Store interface {
	Put(key, value string) error
	Get(key string) (string, error)
}

// Append adds line at the end of the note called title, creating the note if needed.
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
