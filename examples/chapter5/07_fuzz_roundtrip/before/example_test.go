package rle_test

import (
	"fmt"

	rle "github.com/holmeslin/learning-go-to-senior/examples/chapter5/07_fuzz_roundtrip/before"
)

// ANCHOR: example

func Example() {
	for _, in := range []string{"aaabcc", "7", "1a"} {
		enc := rle.Encode(in)
		got, err := rle.Decode(enc)
		fmt.Printf("%q -> %q: round trip ok=%t, decoded %d bytes, err=%v\n", in, enc, got == in, len(got), err)
	}
	// Output:
	// "aaabcc" -> "3a1b2c": round trip ok=true, decoded 6 bytes, err=<nil>
	// "7" -> "17": round trip ok=false, decoded 0 bytes, err=decode: count "17" has no byte after it
	// "1a" -> "111a": round trip ok=false, decoded 111 bytes, err=<nil>
}

// ANCHOR_END: example
