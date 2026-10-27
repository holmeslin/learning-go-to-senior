package form_test

import (
	"fmt"

	form "github.com/holmeslin/learning-go-to-senior/examples/chapter2/05_join_or_first_error/before"
)

// ANCHOR: example

func ExampleValidate() {
	f := form.Form{Name: "", Email: "andy.example.com", Age: "200"}

	fmt.Println(form.Validate(f)) // 第一次送出
	f.Name = "Andy"
	fmt.Println(form.Validate(f)) // 改好名字，再送出
	f.Email = "andy@example.com"
	fmt.Println(form.Validate(f)) // 改好 email，再送出
	// Output:
	// name is required
	// email must contain @
	// age must be between 0 and 150
}

// ANCHOR_END: example
