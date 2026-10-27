// Package app 是 cache 的呼叫端。
package app

import (
	"log"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter1/05_constructor_functional_options/before/cache"
)

// ANCHOR: callers

// NewCaches 建立服務用到的兩個快取。
func NewCaches() (sessions, pages *cache.Cache) {
	sessions = cache.New(1000, 0, nil)
	pages = cache.New(100, 4096, func(key string) { log.Println("evict", key) })
	return sessions, pages
}

// ANCHOR_END: callers
