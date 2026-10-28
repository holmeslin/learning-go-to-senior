package signup_test

import (
	"fmt"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter3/07_di_without_framework/before/signup"
)

// ANCHOR: example

func ExampleRegister() {
	fmt.Println(signup.Register("ann@example.com"))
	// Output:
	// smtp smtp.example.com:25: welcome mail to ann@example.com
	// <nil>
}

func ExampleRegister_again() {
	// 另一個測試也註冊 ann@example.com，卻拿到上一個測試留下的結果。
	fmt.Println(signup.Register("ann@example.com"))
	// Output: already registered
}

// ANCHOR_END: example
