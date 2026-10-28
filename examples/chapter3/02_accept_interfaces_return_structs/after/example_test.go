package audit_test

import (
	"bytes"
	"fmt"
	"io"

	audit "github.com/holmeslin/learning-go-to-senior/examples/chapter3/02_accept_interfaces_return_structs/after"
)

// ANCHOR: discard

func ExampleNew_discard() {
	l := audit.New(io.Discard) // 不想記錄：傳 io.Discard，不需要 nil
	l.Log("login")
	fmt.Println(l.Flush())
	// Output:
	// <nil>
}

// ANCHOR_END: discard

func ExampleLogger_Flush() {
	var buf bytes.Buffer
	l := audit.New(&buf)

	l.Log("login")
	fmt.Printf("%q\n", buf.String())
	if err := l.Flush(); err != nil {
		fmt.Println(err)
	}
	fmt.Printf("%q\n", buf.String())
	// Output:
	// ""
	// "login\n"
}
