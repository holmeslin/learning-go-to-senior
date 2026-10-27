package cacheconf_test

import (
	"fmt"
	"testing"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter1/05_constructor_functional_options/after/cacheconf"
)

// ANCHOR: example
func ExampleNew() {
	// 零值的 Config 就是全部用預設值。
	pages := cacheconf.New(cacheconf.Config{})

	sessions := cacheconf.New(cacheconf.Config{
		Capacity: 2,
		OnEvict:  func(key string) { fmt.Println("evict", key) },
	})
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
	c := cacheconf.New(cacheconf.Config{})
	for i := range 101 {
		c.Set(fmt.Sprint(i), "v")
	}
	if _, ok := c.Get("0"); ok {
		t.Error("預設容量 100：第 101 筆寫入後，最舊的一筆應該被淘汰")
	}
}

func TestMaxValueLen(t *testing.T) {
	c := cacheconf.New(cacheconf.Config{MaxValueLen: 3})
	if c.Set("k", "abcd") {
		t.Error("超過長度限制的值應該被拒絕")
	}
	if !c.Set("k", "abc") {
		t.Error("剛好等於限制的值應該被接受")
	}
}
