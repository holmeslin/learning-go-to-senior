// Package crawl 依序抓取一批網址，把結果交給呼叫端。
package crawl

import "context"

// Fetch 抓取 url 的內容。
type Fetch func(ctx context.Context, url string) string

// ANCHOR: pages

// Pages 依序抓取 urls，把內容送進回傳的 channel；全部抓完後關閉 channel。
func Pages(ctx context.Context, urls []string, fetch Fetch) <-chan string {
	out := make(chan string)
	go func() {
		defer close(out)
		for _, u := range urls {
			out <- fetch(ctx, u)
		}
	}()
	return out
}

// ANCHOR_END: pages
