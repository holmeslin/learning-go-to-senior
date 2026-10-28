package commands_test

import (
	"fmt"

	commands "github.com/holmeslin/learning-go-to-senior/examples/chapter5/10_flaky_tests/after"
)

// ANCHOR: example

func ExampleRegistry() {
	r := commands.New()
	fmt.Println(r.Register("echo", echo))
	fmt.Println(r.Register("echo", echo))
	_, err := commands.New().Run("echo", "hi")
	fmt.Println(err)
	// Output:
	// <nil>
	// register "echo": already registered
	// run "echo": unknown command
}

// ANCHOR_END: example
