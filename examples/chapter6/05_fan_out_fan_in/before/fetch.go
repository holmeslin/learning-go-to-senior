// Package crawl 抓取一批網址的頁面：每個網址一個 goroutine，結果送進 channel 再收回來。
package crawl

// Page 是抓回來的一頁。
type Page struct {
	URL   string
	Title string
}

// Fetcher 抓取 url 對應的頁面。
type Fetcher func(url string) Page

// ANCHOR: fetchall

// FetchAll 為每個網址開一個 goroutine 抓取，結果依抵達的順序收集，不保證與 urls 同序。
func FetchAll(urls []string, fetch Fetcher) []Page {
	results := make(chan Page)
	for _, url := range urls {
		go func() { results <- fetch(url) }()
	}

	var pages []Page
	for range urls {
		pages = append(pages, <-results)
	}
	return pages
}

// ANCHOR_END: fetchall
