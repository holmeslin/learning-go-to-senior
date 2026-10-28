package rle_test

import (
	"testing"

	rle "github.com/holmeslin/learning-go-to-senior/examples/chapter5/07_fuzz_roundtrip/before"
)

// ANCHOR: table

func TestEncode(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", ""},
		{"a", "1a"},
		{"abc", "1a1b1c"},
		{"aaabbc", "3a2b1c"},
		{"zzzzzzzzzzzz", "12z"},
	}
	for _, tt := range tests {
		if got := rle.Encode(tt.in); got != tt.want {
			t.Errorf("Encode(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestRoundTrip(t *testing.T) {
	for _, in := range []string{"", "a", "abc", "aaabbc", "zzzzzzzzzzzz", "hello, world", "héllo"} {
		got, err := rle.Decode(rle.Encode(in))
		if err != nil {
			t.Errorf("Decode(Encode(%q)) error = %v, want nil", in, err)
			continue
		}
		if got != in {
			t.Errorf("Decode(Encode(%q)) = %q, want %q", in, got, in)
		}
	}
}

// ANCHOR_END: table
