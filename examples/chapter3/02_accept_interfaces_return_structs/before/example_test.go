package audit_test

import (
	"fmt"

	audit "github.com/holmeslin/learning-go-to-senior/examples/chapter3/02_accept_interfaces_return_structs/before"
)

// ANCHOR: typednil

func ExampleNew_disabled() {
	defer func() { fmt.Println("recovered:", recover()) }()

	l := audit.New(nil) // 文件說：f 為 nil 表示不記錄，回傳 nil
	fmt.Println(l == nil)
	fmt.Printf("%T\n", l)
	if l != nil {
		l.Log("login")
	}
	// Output:
	// false
	// *audit.bufLogger
	// recovered: runtime error: invalid memory address or nil pointer dereference
}

// ANCHOR_END: typednil
