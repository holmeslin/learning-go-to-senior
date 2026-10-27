package cache_test

import (
	"fmt"
	"testing"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter1/05_constructor_functional_options/after/cache"
)

// ANCHOR: example

func ExampleNew() {
	// 全部用預設值：容量 100，寫第 101 筆時最舊的一筆被淘汰。
	pages, _ := cache.New()
	for i := range 101 {
		pages.Set(fmt.Sprint(i), "v")
	}
	_, first := pages.Get("0")
	_, second := pages.Get("1")
	fmt.Println(first, second)

	// 只覆寫需要的選項，順序不重要，也看得出每個值的意思。
	sessions, _ := cache.New(
		cache.WithCapacity(2),
		cache.WithOnEvict(func(key string) { fmt.Println("evict", key) }),
	)
	sessions.Set("a", "1")
	sessions.Set("b", "2")
	sessions.Set("c", "3")

	// 選項可以在建構時檢查：容量小於 1 沒有意義，New 直接回傳錯誤。
	_, err := cache.New(cache.WithCapacity(0))
	fmt.Println(err)
	// Output:
	// false true
	// evict a
	// cache: capacity must be at least 1, got 0
}

// ANCHOR_END: example

func TestWithCapacityRejectsInvalid(t *testing.T) {
	for _, n := range []int{0, -1} {
		c, err := cache.New(cache.WithCapacity(n))
		if err == nil || c != nil {
			t.Errorf("WithCapacity(%d): got (%v, %v), want nil cache and an error", n, c, err)
		}
	}
	if _, err := cache.New(cache.WithCapacity(1)); err != nil {
		t.Errorf("WithCapacity(1): unexpected error %v", err)
	}
}

func TestNewDefaults(t *testing.T) {
	c, err := cache.New()
	if err != nil {
		t.Fatal(err)
	}
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
	c, err := cache.New(cache.WithMaxValueLen(3))
	if err != nil {
		t.Fatal(err)
	}
	if c.Set("k", "abcd") {
		t.Error("超過長度限制的值應該被拒絕")
	}
	if !c.Set("k", "abc") {
		t.Error("剛好等於限制的值應該被接受")
	}
}
