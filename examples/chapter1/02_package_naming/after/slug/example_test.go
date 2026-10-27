package slug_test

import (
	"fmt"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter1/02_package_naming/after/slug"
)

func ExampleMake() {
	fmt.Println(slug.Make("Hello Go World"))
	// Output: hello-go-world
}
