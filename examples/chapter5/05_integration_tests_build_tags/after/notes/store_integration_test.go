//go:build integration

package notes_test

import (
	"errors"
	"path/filepath"
	"slices"
	"testing"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter5/05_integration_tests_build_tags/after/notes"
)

// ANCHOR: test

func TestStorePersistsAcrossReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.db")

	store, err := notes.Open(t.Context(), path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	for _, text := range []string{"buy milk", "  call ann "} {
		if _, err := store.Add(t.Context(), text); err != nil {
			t.Fatalf("Add(%q) error = %v", text, err)
		}
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	reopened, err := notes.Open(t.Context(), path)
	if err != nil {
		t.Fatalf("reopen error = %v", err)
	}
	defer reopened.Close()
	got, err := reopened.List(t.Context())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if want := []string{"buy milk", "call ann"}; !slices.Equal(got, want) {
		t.Errorf("List() after reopen = %q, want %q", got, want)
	}
}

// ANCHOR_END: test

func TestStoreRejectsEmptyNote(t *testing.T) {
	store, err := notes.Open(t.Context(), filepath.Join(t.TempDir(), "notes.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()

	if _, err := store.Add(t.Context(), "   "); !errors.Is(err, notes.ErrEmpty) {
		t.Errorf("Add(blank) error = %v, want %v", err, notes.ErrEmpty)
	}
	got, err := store.List(t.Context())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(got) != 0 {
		t.Errorf("List() = %q, want nothing stored", got)
	}
}
