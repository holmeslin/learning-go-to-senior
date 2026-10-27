package text_test

import (
	"fmt"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter1/02_package_naming/after/text"
)

func ExampleTruncate() {
	fmt.Println(text.Truncate("邁向資深 Go 工程師", 4))
	fmt.Println(text.Truncate("短", 4))
	// Output:
	// 邁向資深…
	// 短
}
