package crawl_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	crawl "github.com/holmeslin/learning-go-to-senior/examples/chapter6/05_fan_out_fan_in/after"
)

// slowFirst 讓越前面的網址越慢：/a 要 3 秒、/b 2 秒、/c 1 秒，完成的順序剛好和輸入相反。
// 只能在 synctest 的 bubble 裡用：那裡的時間是假的，所以不用真的等。
func slowFirst(url string) crawl.Page {
	delay := map[string]time.Duration{"/a": 3 * time.Second, "/b": 2 * time.Second, "/c": time.Second}[url]
	time.Sleep(delay)
	return crawl.Page{URL: url, Title: "title" + url}
}

// urlsOf 取出每一頁的網址。
func urlsOf(pages []crawl.Page) []string {
	urls := make([]string, len(pages))
	for i, p := range pages {
		urls[i] = p.URL
	}
	return urls
}

// ANCHOR: order

func TestFetchAllKeepsInputOrder(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		urls := []string{"/a", "/b", "/c"}

		got := urlsOf(crawl.FetchAll(urls, slowFirst))

		if !slices.Equal(got, urls) {
			t.Errorf("FetchAll() = %v, want %v", got, urls)
		}
	})
}

// ANCHOR_END: order

func TestFetchAllNoURLs(t *testing.T) {
	if got := crawl.FetchAll(nil, slowFirst); len(got) != 0 {
		t.Errorf("FetchAll(nil) = %v, want empty", got)
	}
}

func TestStreamDeliversInCompletionOrderThenCloses(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var got []crawl.Page
		for p := range crawl.Stream([]string{"/a", "/b", "/c"}, slowFirst) {
			got = append(got, p)
		}

		// range 能結束，代表 channel 被關閉了；順序是完成的順序，不是輸入的順序。
		if want := []string{"/c", "/b", "/a"}; !slices.Equal(urlsOf(got), want) {
			t.Errorf("Stream() delivered %v, want %v", urlsOf(got), want)
		}
	})
}

// ANCHOR: stream

func TestStreamDeliversBeforeAllDone(t *testing.T) {
	release := make(chan struct{})
	fetch := func(url string) crawl.Page {
		if url == "/slow" {
			<-release // 卡住，直到測試放行
		}
		return crawl.Page{URL: url}
	}

	pages := crawl.Stream([]string{"/fast", "/slow"}, fetch)

	// /slow 還沒完成，/fast 就已經可以處理：slice 版本要等全部完成才拿得到。
	if p := <-pages; p.URL != "/fast" {
		t.Fatalf("first page = %q, want %q", p.URL, "/fast")
	}
	close(release)
	if p := <-pages; p.URL != "/slow" {
		t.Errorf("second page = %q, want %q", p.URL, "/slow")
	}
	if _, ok := <-pages; ok {
		t.Error("pages still open after all fetches finished, want closed")
	}
}

// ANCHOR_END: stream

// ANCHOR: example

func ExampleFetchAll() {
	fetch := func(url string) crawl.Page { return crawl.Page{URL: url, Title: "title" + url} }

	for _, p := range crawl.FetchAll([]string{"/a", "/b", "/c"}, fetch) {
		fmt.Println(p.URL, p.Title)
	}
	// Output:
	// /a title/a
	// /b title/b
	// /c title/c
}

// ANCHOR_END: example

func ExampleStream() {
	fetch := func(url string) crawl.Page { return crawl.Page{URL: url, Title: "title" + url} }

	var urls []string
	for p := range crawl.Stream([]string{"/a", "/b", "/c"}, fetch) {
		urls = append(urls, p.URL)
	}
	slices.Sort(urls) // 抵達順序不固定，排序後才能印出固定的輸出
	fmt.Println(strings.Join(urls, " "))
	// Output:
	// /a /b /c
}
