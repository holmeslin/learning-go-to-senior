package rle_test

import (
	"fmt"

	rle "github.com/holmeslin/learning-go-to-senior/examples/chapter5/07_fuzz_roundtrip/after"
)

// ANCHOR: example

func Example() {
	for _, in := range []string{"aaabcc", "7", "1a"} {
		enc := rle.Encode(in)
		got, err := rle.Decode(enc)
		fmt.Printf("%q -> %q: round trip ok=%t, decoded %d bytes, err=%v\n", in, enc, got == in, len(got), err)
	}
	// Output:
	// "aaabcc" -> "3:a1:b2:c": round trip ok=true, decoded 6 bytes, err=<nil>
	// "7" -> "1:7": round trip ok=true, decoded 1 bytes, err=<nil>
	// "1a" -> "1:11:a": round trip ok=true, decoded 2 bytes, err=<nil>
}

// ANCHOR_END: example
