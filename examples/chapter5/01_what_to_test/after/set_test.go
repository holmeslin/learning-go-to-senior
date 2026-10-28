package tagset_test

import (
	"slices"
	"testing"

	tagset "github.com/holmeslin/learning-go-to-senior/examples/chapter5/01_what_to_test/after"
)

// ANCHOR: test

func TestAddNormalizesTags(t *testing.T) {
	s := tagset.New()
	s.Add("  Go ")

	if got, want := s.List(), []string{"go"}; !slices.Equal(got, want) {
		t.Errorf("List() = %q, want %q", got, want)
	}
}

func TestAddReportsWhetherTagIsNew(t *testing.T) {
	s := tagset.New()

	if !s.Add("go") {
		t.Error(`Add("go") = false, want true for a new tag`)
	}
	if s.Add("GO") {
		t.Error(`Add("GO") = true, want false: same tag after normalizing`)
	}
	if got, want := s.Len(), 1; got != want {
		t.Errorf("Len() = %d, want %d", got, want)
	}
}

func TestAddIgnoresBlankTags(t *testing.T) {
	s := tagset.New()

	if s.Add("   ") {
		t.Error(`Add("   ") = true, want false`)
	}
	if got, want := s.Len(), 0; got != want {
		t.Errorf("Len() = %d, want %d", got, want)
	}
}

func TestContainsNormalizesTags(t *testing.T) {
	s := tagset.New()
	s.Add("Go")

	for _, tag := range []string{"go", "Go", " GO "} {
		if !s.Contains(tag) {
			t.Errorf("Contains(%q) = false, want true", tag)
		}
	}
	if s.Contains("rust") {
		t.Error(`Contains("rust") = true, want false`)
	}
}

func TestListIsSorted(t *testing.T) {
	s := tagset.New()
	for _, tag := range []string{"rust", "go", "zig"} {
		s.Add(tag)
	}

	if got, want := s.List(), []string{"go", "rust", "zig"}; !slices.Equal(got, want) {
		t.Errorf("List() = %q, want %q", got, want)
	}
}

func TestListReturnsACopy(t *testing.T) {
	s := tagset.New()
	s.Add("go")

	s.List()[0] = "changed"

	if got, want := s.List(), []string{"go"}; !slices.Equal(got, want) {
		t.Errorf("List() = %q after modifying a previous result, want %q", got, want)
	}
}

// ANCHOR_END: test
