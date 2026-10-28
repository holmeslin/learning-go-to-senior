package crawl_test

import (
	"context"
	"fmt"

	crawl "github.com/holmeslin/learning-go-to-senior/examples/chapter6/04_cancellation_context/before"
)

// ANCHOR: example

func ExamplePages() {
	urls := []string{"/a", "/b", "/c", "/d", "/e"}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fetched := 0 // 只有抓取的 goroutine 會寫；channel 關閉之後讀才安全
	fetch := func(_ context.Context, url string) string {
		fetched++
		if url == "/b" {
			cancel() // 使用者在第二筆抓到一半時離開
		}
		return "page " + url
	}

	for range crawl.Pages(ctx, urls, fetch) {
	}
	fmt.Printf("fetched %d of %d urls\n", fetched, len(urls))
	// Output:
	// fetched 5 of 5 urls
}

// ANCHOR_END: example
