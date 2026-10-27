package size_test

import (
	"fmt"

	size "github.com/holmeslin/learning-go-to-senior/examples/chapter1/08_deprecation_and_compatibility/after"
)

// ANCHOR: example

func ExampleParse() {
	n, err := size.Parse("10MiB")
	fmt.Println(n, err)
	// Output: 10485760 <nil>
}

// ANCHOR_END: example
