package tagset

import (
	"slices"
	"testing"
)

// ANCHOR: test

func TestAddStoresNormalizedTag(t *testing.T) {
	s := New()
	s.Add("  Go ")

	if got, want := s.items, []string{"go"}; !slices.Equal(got, want) {
		t.Errorf("items = %q, want %q", got, want)
	}
}

func TestAddKeepsInsertionOrder(t *testing.T) {
	s := New()
	s.Add("rust")
	s.Add("go")

	if got, want := s.items, []string{"rust", "go"}; !slices.Equal(got, want) {
		t.Errorf("items = %q, want %q", got, want)
	}
}

func TestAddDuplicateLeavesItemsUnchanged(t *testing.T) {
	s := New()
	s.Add("rust")
	s.Add("go")
	s.Add("GO")

	if got, want := s.items, []string{"rust", "go"}; !slices.Equal(got, want) {
		t.Errorf("items = %q, want %q", got, want)
	}
}

func TestIndexOf(t *testing.T) {
	s := &Set{items: []string{"rust", "go"}}

	if got, want := s.indexOf("go"), 1; got != want {
		t.Errorf("indexOf(%q) = %d, want %d", "go", got, want)
	}
	if got, want := s.indexOf("zig"), -1; got != want {
		t.Errorf("indexOf(%q) = %d, want %d", "zig", got, want)
	}
}

func TestListIsSorted(t *testing.T) {
	s := New()
	s.Add("rust")
	s.Add("go")

	if got, want := s.List(), []string{"go", "rust"}; !slices.Equal(got, want) {
		t.Errorf("List() = %q, want %q", got, want)
	}
}

// ANCHOR_END: test
