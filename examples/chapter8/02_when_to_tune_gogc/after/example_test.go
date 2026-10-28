package roster_test

import (
	"fmt"

	roster "github.com/holmeslin/learning-go-to-senior/examples/chapter8/02_when_to_tune_gogc/after"
)

// ANCHOR: example

func ExampleRoster_Handle() {
	r := roster.Load(10)
	// 99 不在名冊裡，直接略過。
	fmt.Printf("%s\n", r.Handle([]int{3, 4, 99, 8}))
	// Output:
	// 3,member-3,free
	// 4,member-4,pro
	// 8,member-8,team
}

// ANCHOR_END: example
