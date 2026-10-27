package tally_test

import (
	"os"

	tally "github.com/holmeslin/learning-go-to-senior/examples/chapter1/06_zero_value_or_constructor/after"
)

// ANCHOR: example
func ExampleCounter() {
	var c tally.Counter // 不用建構函式，零值直接可用
	c.Add("/home")
	c.Add("/home")
	c.Add("/about")

	r := tally.NewReporter(os.Stdout) // 需要 io.Writer，只能用建構函式
	r.Report(&c)
	// Output:
	// /about 1
	// /home 2
}

// ANCHOR_END: example
