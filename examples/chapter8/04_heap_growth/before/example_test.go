package ingest_test

import (
	"fmt"

	ingest "github.com/holmeslin/learning-go-to-senior/examples/chapter8/04_heap_growth/before"
)

// ANCHOR: example

func ExampleSeen() {
	payload := make([]byte, 64<<10) // 一筆 64 KiB 的通知
	var s ingest.Seen
	if err := s.Add(payload); err != nil {
		fmt.Println(err)
		return
	}
	id := s.IDs()[0]
	fmt.Printf("len=%d cap=%d\n", len(id), cap(id))
	// Output:
	// len=16 cap=65536
}

// ANCHOR_END: example
