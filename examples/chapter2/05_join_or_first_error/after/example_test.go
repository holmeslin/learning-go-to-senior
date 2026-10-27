package form_test

import (
	"errors"
	"fmt"

	form "github.com/holmeslin/learning-go-to-senior/examples/chapter2/05_join_or_first_error/after"
)

// ANCHOR: example

func ExampleValidate() {
	err := form.Validate(form.Form{Name: "", Email: "andy.example.com", Age: "200"})
	fmt.Println(err)
	// Output:
	// name: required
	// email: must contain @
	// age: must be between 0 and 150
}

// ANCHOR_END: example

// ANCHOR: inspect

func ExampleValidate_inspect() {
	err := form.Validate(form.Form{Name: "", Email: "andy.example.com", Age: "200"})

	fmt.Println(errors.Is(err, form.ErrRequired)) // 找得到任何一個

	if fe, ok := errors.AsType[*form.FieldError](err); ok { // 只回傳第一個符合的
		fmt.Println("first field:", fe.Field)
	}

	if joined, ok := err.(interface{ Unwrap() []error }); ok { // 要逐一處理，就自己展開
		for _, e := range joined.Unwrap() {
			fmt.Println("-", e)
		}
	}
	// Output:
	// true
	// first field: name
	// - name: required
	// - email: must contain @
	// - age: must be between 0 and 150
}

// ANCHOR_END: inspect
