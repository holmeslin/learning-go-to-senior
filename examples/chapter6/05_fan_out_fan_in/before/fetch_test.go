package crawl_test

import (
	"slices"
	"strings"
	"testing"

	crawl "github.com/holmeslin/learning-go-to-senior/examples/chapter6/05_fan_out_fan_in/before"
)

// ANCHOR: test

func TestFetchAll(t *testing.T) {
	urls := []string{"/a", "/b", "/c"}
	fetch := func(url string) crawl.Page { return crawl.Page{URL: url, Title: "title" + url} }

	got := crawl.FetchAll(urls, fetch)

	// 抵達順序不固定，所以只能排序後比較：測試確認得了「全部都在」，確認不了順序。
	slices.SortFunc(got, func(a, b crawl.Page) int { return strings.Compare(a.URL, b.URL) })
	want := []crawl.Page{{URL: "/a", Title: "title/a"}, {URL: "/b", Title: "title/b"}, {URL: "/c", Title: "title/c"}}
	if !slices.Equal(got, want) {
		t.Errorf("FetchAll() = %v, want %v", got, want)
	}
}

// ANCHOR_END: test
