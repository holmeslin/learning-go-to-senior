// Package tagset keeps a set of tags that ignore case and surrounding spaces.
package tagset

import (
	"slices"
	"strings"
)

// ANCHOR: set

// Set is a set of normalized tags. The zero value is not ready for use;
// call New.
type Set struct {
	items []string
}

// New returns an empty Set.
func New() *Set {
	return &Set{}
}

// Add trims and lowercases tag, then adds it. It reports whether the tag was
// new; blank tags are ignored.
func (s *Set) Add(tag string) bool {
	tag = normalize(tag)
	if tag == "" || s.indexOf(tag) >= 0 {
		return false
	}
	s.items = append(s.items, tag)
	return true
}

// Contains reports whether tag, after trimming and lowercasing, is in the set.
func (s *Set) Contains(tag string) bool {
	return s.indexOf(normalize(tag)) >= 0
}

// Len returns the number of tags.
func (s *Set) Len() int {
	return len(s.items)
}

// List returns the tags in alphabetical order. The caller may modify the
// returned slice.
func (s *Set) List() []string {
	out := slices.Clone(s.items)
	slices.Sort(out)
	return out
}

func (s *Set) indexOf(tag string) int {
	return slices.Index(s.items, tag)
}

func normalize(tag string) string {
	return strings.ToLower(strings.TrimSpace(tag))
}

// ANCHOR_END: set
