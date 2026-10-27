package cache_test

import (
	"fmt"
	"testing"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter1/05_constructor_functional_options/after/cache"
)

// ANCHOR: example
func ExampleNew() {
	// 全部用預設值。
	pages := cache.New()

	// 只覆寫需要的選項，順序不重要，也看得出每個值的意思。
	sessions := cache.New(
		cache.WithCapacity(2),
		cache.WithOnEvict(func(key string) { fmt.Println("evict", key) }),
	)
	sessions.Set("a", "1")
	sessions.Set("b", "2")
	sessions.Set("c", "3")

	_, ok := pages.Get("a")
	fmt.Println(ok)
	// Output:
	// evict a
	// false
}

// ANCHOR_END: example

func TestNewDefaults(t *testing.T) {
	c := cache.New()
	for i := range 101 {
		c.Set(fmt.Sprint(i), "v")
	}
	if _, ok := c.Get("0"); ok {
		t.Error("預設容量 100：第 101 筆寫入後，最舊的一筆應該被淘汰")
	}
	if _, ok := c.Get("1"); !ok {
		t.Error("第二舊的一筆不該被淘汰")
	}
}

func TestWithMaxValueLen(t *testing.T) {
	c := cache.New(cache.WithMaxValueLen(3))
	if c.Set("k", "abcd") {
		t.Error("超過長度限制的值應該被拒絕")
	}
	if !c.Set("k", "abc") {
		t.Error("剛好等於限制的值應該被接受")
	}
}
