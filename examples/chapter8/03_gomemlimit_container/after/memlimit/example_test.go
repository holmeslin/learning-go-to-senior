package memlimit_test

import (
	"fmt"
	"testing/fstest"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter8/03_gomemlimit_container/after/memlimit"
)

// ANCHOR: example

func ExampleLimit() {
	for _, content := range []string{"268435456\n", "max\n", "abc\n"} {
		fsys := fstest.MapFS{"memory.max": {Data: []byte(content)}}
		limit, err := memlimit.Limit(fsys)
		fmt.Println(limit, err)
	}
	_, err := memlimit.Limit(fstest.MapFS{})
	fmt.Println(err)
	// Output:
	// 241591911 <nil>
	// 0 no memory limit
	// 0 parse memory.max "abc": strconv.ParseInt: parsing "abc": invalid syntax
	// read memory limit: open memory.max: file does not exist
}

// ANCHOR_END: example
