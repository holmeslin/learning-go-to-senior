package rle_test

import (
	"testing"

	rle "github.com/holmeslin/learning-go-to-senior/examples/chapter5/07_fuzz_roundtrip/after"
)

func TestEncode(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", ""},
		{"a", "1:a"},
		{"abc", "1:a1:b1:c"},
		{"aaabbc", "3:a2:b1:c"},
		{"zzzzzzzzzzzz", "12:z"},
		{"7", "1:7"},
		{"1a", "1:11:a"},
	}
	for _, tt := range tests {
		if got := rle.Encode(tt.in); got != tt.want {
			t.Errorf("Encode(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestDecodeRejectsMalformedInput(t *testing.T) {
	for _, in := range []string{"a", "3", "1:", "1a", "99999999999:a"} {
		if got, err := rle.Decode(in); err == nil {
			t.Errorf("Decode(%q) = %q, nil, want an error", in, got)
		}
	}
}

// ANCHOR: fuzz

// FuzzRoundTrip 檢查一條性質：不論輸入是什麼，編碼再解碼都要得到原本的字串。
// 性質只涵蓋單一連續段不超過 rle 的 maxRun（約一百萬位元組）的字串，更長的連續段 Decode 會拒絕。
func FuzzRoundTrip(f *testing.F) {
	for _, seed := range []string{"", "a", "abc", "aaabbc", "zzzzzzzzzzzz", "hello, world", "héllo"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, s string) {
		enc := rle.Encode(s)
		got, err := rle.Decode(enc)
		if err != nil {
			t.Fatalf("Decode(Encode(%q)) error = %v (encoded as %q), want nil", s, err, enc)
		}
		if got != s {
			t.Errorf("Decode(Encode(%q)) = %q (encoded as %q), want %q", s, got, enc, s)
		}
	})
}

// ANCHOR_END: fuzz
