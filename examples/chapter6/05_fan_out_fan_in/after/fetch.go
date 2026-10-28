// Package crawl 抓取一批網址的頁面，提供兩種 fan-in：依索引寫進 slice，或串流給呼叫端。
package crawl

import "sync"

// Page 是抓回來的一頁。
type Page struct {
	URL   string
	Title string
}

// Fetcher 抓取 url 對應的頁面。
type Fetcher func(url string) Page

// ANCHOR: fetchall

// FetchAll 為每個網址開一個 goroutine 抓取，全部完成後回傳，結果與 urls 同序。
func FetchAll(urls []string, fetch Fetcher) []Page {
	pages := make([]Page, len(urls))
	var wg sync.WaitGroup
	for i, url := range urls {
		wg.Go(func() { pages[i] = fetch(url) }) // 每個 goroutine 只寫自己的那一格
	}
	wg.Wait()
	return pages
}

// ANCHOR_END: fetchall

// ANCHOR: stream

// Stream 為每個網址開一個 goroutine 抓取，抓完一頁就送出一頁，順序是完成的順序。
// 全部完成後 channel 會被關閉；呼叫端必須把它讀到關閉為止，否則還沒送出的 goroutine 會卡住。
func Stream(urls []string, fetch Fetcher) <-chan Page {
	results := make(chan Page)
	var wg sync.WaitGroup
	for _, url := range urls {
		wg.Go(func() { results <- fetch(url) })
	}
	go func() {
		wg.Wait() // 只有在全部送完之後，才由這個 goroutine 關閉
		close(results)
	}()
	return results
}

// ANCHOR_END: stream
