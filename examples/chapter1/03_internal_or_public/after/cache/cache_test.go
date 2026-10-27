package cache_test

import (
	"testing"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter1/03_internal_or_public/after/cache"
)

func TestStats(t *testing.T) {
	c := cache.New[string, int](2)
	c.Put("a", 1)
	c.Get("a")
	c.Get("missing")

	if got, want := c.Stats(), (cache.Stats{Hits: 1, Misses: 1}); got != want {
		t.Errorf("Stats() = %+v, want %+v", got, want)
	}
}

func TestPutOverwritesWithoutEvicting(t *testing.T) {
	c := cache.New[string, int](2)
	c.Put("a", 1)
	c.Put("b", 2)
	c.Put("a", 10)

	if got, ok := c.Get("a"); !ok || got != 10 {
		t.Errorf("Get(a) = %d, %v; want 10, true", got, ok)
	}
	if _, ok := c.Get("b"); !ok {
		t.Error("overwriting an existing key must not evict another one")
	}
}

func TestCapacityBelowOneHoldsOneItem(t *testing.T) {
	c := cache.New[string, int](0)
	c.Put("a", 1)
	c.Put("b", 2)

	if _, ok := c.Get("a"); ok {
		t.Error("a should have been evicted")
	}
	if _, ok := c.Get("b"); !ok {
		t.Error("b should be cached")
	}
}
