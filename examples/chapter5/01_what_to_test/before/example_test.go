package tagset_test

import (
	"fmt"

	tagset "github.com/holmeslin/learning-go-to-senior/examples/chapter5/01_what_to_test/before"
)

// ANCHOR: example

func ExampleSet_Contains() {
	s := tagset.New()
	s.Add("Go")

	fmt.Println(s.Contains("go"), s.Contains("Go"))
	// Output: true false
}

// ANCHOR_END: example
