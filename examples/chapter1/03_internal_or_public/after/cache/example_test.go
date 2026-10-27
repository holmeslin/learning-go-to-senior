package cache_test

import (
	"fmt"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter1/03_internal_or_public/after/cache"
)

// ANCHOR: example

func ExampleCache() {
	c := cache.New[string, int](2)
	c.Put("a", 1)
	c.Put("b", 2)
	c.Get("a")    // a 剛被用過，b 變成最久沒用的
	c.Put("c", 3) // 超過容量，淘汰 b

	_, okA := c.Get("a")
	_, okB := c.Get("b")
	fmt.Println(okA, okB)
	// Output:
	// true false
}

// ANCHOR_END: example
