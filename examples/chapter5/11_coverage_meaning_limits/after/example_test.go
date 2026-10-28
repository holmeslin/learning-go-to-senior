package split_test

import (
	"fmt"

	split "github.com/holmeslin/learning-go-to-senior/examples/chapter5/11_coverage_meaning_limits/after"
)

// ANCHOR: example

func ExampleSplit() {
	fmt.Println(split.Split(100, 1, 1, 1))
	fmt.Println(split.Split(100, 1, -1))
	fmt.Println(split.Split(-100, 1, 1))
	// Output:
	// [34 33 33] <nil>
	// [] weight must be positive
	// [] split -100: total must not be negative
}

// ANCHOR_END: example
