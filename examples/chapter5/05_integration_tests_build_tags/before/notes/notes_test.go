package notes_test

import (
	"errors"
	"path/filepath"
	"slices"
	"testing"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter5/05_integration_tests_build_tags/before/notes"
)

// ANCHOR: tests

func TestNormalize(t *testing.T) {
	got, err := notes.Normalize("  call ann \n")
	if err != nil || got != "call ann" {
		t.Errorf("Normalize() = %q, %v; want %q, nil", got, err, "call ann")
	}
	if _, err := notes.Normalize("   "); !errors.Is(err, notes.ErrEmpty) {
		t.Errorf("Normalize(blank) error = %v, want %v", err, notes.ErrEmpty)
	}
}

// TestStoreAddList 開真的檔案，卻和上面的單元測試放在同一個檔案。
func TestStoreAddList(t *testing.T) {
	store, err := notes.Open(t.Context(), filepath.Join(t.TempDir(), "notes.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()
	if _, err := store.Add(t.Context(), "buy milk"); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	got, err := store.List(t.Context())
	if err != nil || !slices.Equal(got, []string{"buy milk"}) {
		t.Errorf("List() = %q, %v; want [buy milk]", got, err)
	}
}

// ANCHOR_END: tests
