package slugtool_test

import (
	"fmt"

	slugtool "github.com/holmeslin/learning-go-to-senior/examples/chapter12/07_release_and_versioning/before"
)

// v1 的 Slug 只回傳字串；空白的標題會得到空字串，呼叫端不一定會發現。
func Example() {
	fmt.Printf("%q\n", slugtool.Slug("Hello Go World"))
	fmt.Printf("%q\n", slugtool.Slug("   "))
	// Output:
	// "hello-go-world"
	// ""
}
