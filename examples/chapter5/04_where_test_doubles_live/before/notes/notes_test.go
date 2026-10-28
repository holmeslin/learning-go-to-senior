package notes

import (
	"errors"
	"testing"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter5/04_where_test_doubles_live/before/kv"
)

func TestAppend(t *testing.T) {
	t.Parallel()
	store := newFakeStore()

	for _, line := range []string{"buy milk", "call Ann"} {
		if err := Append(store, "todo", line); err != nil {
			t.Fatalf("Append(%q) error = %v", line, err)
		}
	}

	got, err := store.Get("todo")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if want := "buy milk\ncall Ann"; got != want {
		t.Errorf("note = %q, want %q", got, want)
	}
}

func TestAppendRejectsInvalidTitle(t *testing.T) {
	t.Parallel()
	err := Append(newFakeStore(), "work/todo", "buy milk")
	if !errors.Is(err, kv.ErrInvalidKey) {
		t.Errorf("Append() error = %v, want kv.ErrInvalidKey", err)
	}
}
