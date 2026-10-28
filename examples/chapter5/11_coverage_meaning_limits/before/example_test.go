package split_test

import (
	"fmt"

	split "github.com/holmeslin/learning-go-to-senior/examples/chapter5/11_coverage_meaning_limits/before"
)

// ANCHOR: example

func ExampleSplit() {
	fmt.Println(split.Split(100, 1, 1, 1))
	fmt.Println(split.Split(100, 1, -1)) // 預期 weight must be positive
	fmt.Println(split.Split(-100, 1, 1)) // 預期 total must not be negative
	// Output:
	// [34 33 33] <nil>
	// [] no weights
	// [0 0] <nil>
}

// ANCHOR_END: example
