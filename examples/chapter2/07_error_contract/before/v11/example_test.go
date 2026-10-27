package v11_test

import (
	"errors"
	"fmt"

	v1 "github.com/holmeslin/learning-go-to-senior/examples/chapter2/07_error_contract/before/v1"
	v11 "github.com/holmeslin/learning-go-to-senior/examples/chapter2/07_error_contract/before/v11"
)

// ANCHOR: example

func Example() {
	// 呼叫端的判斷：找不到時要走另一條路。兩次都是同一行，只有 import 不同。
	_, err := v1.Find(42)
	fmt.Println("v1:  ", errors.Is(err, v1.ErrNotFound))

	_, err = v11.Find(42)
	fmt.Println("v1.1:", errors.Is(err, v11.ErrNotFound))
	// Output:
	// v1:   true
	// v1.1: false
}

// ANCHOR_END: example
