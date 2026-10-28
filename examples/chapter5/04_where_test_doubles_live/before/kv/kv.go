// Package kv is a tiny key-value store that keeps each value in its own file.
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
	// ErrNotFound means no value is stored under the key.
	ErrNotFound = errors.New("kv: key not found")
	// ErrInvalidKey means the key is not allowed: keys are file names, so they must be flat.
	ErrInvalidKey = errors.New("kv: invalid key")
)

// ANCHOR: validkey

// ValidKey reports whether key may be used with a Store.
// A key must be a non-empty file name: no slash, and not "." or "..".
func ValidKey(key string) bool {
	return fs.ValidPath(key) && key != "." && !strings.Contains(key, "/")
}

// ANCHOR_END: validkey

// DirStore keeps each value in a file named after its key, inside one directory.
type DirStore struct {
	dir string
}

// NewDirStore returns a DirStore that uses dir, which must already exist.
func NewDirStore(dir string) *DirStore {
	return &DirStore{dir: dir}
}

// Put stores value under key, replacing any previous value.
// It returns ErrInvalidKey if key is not valid.
func (s *DirStore) Put(key, value string) error {
	if !ValidKey(key) {
		return ErrInvalidKey
	}
	if err := os.WriteFile(filepath.Join(s.dir, key), []byte(value), 0o600); err != nil {
		return fmt.Errorf("kv: put: %w", err)
	}
	return nil
}

// Get returns the value stored under key.
// It returns ErrInvalidKey if key is not valid and ErrNotFound if nothing is stored.
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
