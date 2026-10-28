package accesslog_test

import (
	"fmt"
	"testing"

	accesslog "github.com/holmeslin/learning-go-to-senior/examples/chapter7/05_common_hotspots/before"
)

// sample 產生 n 筆紀錄，每四筆有一筆是失敗請求，路徑都已經是小寫。
func sample(n int) []accesslog.Entry {
	entries := make([]accesslog.Entry, n)
	for i := range entries {
		status := 200
		if i%4 == 0 {
			status = 404
		}
		entries[i] = accesslog.Entry{Method: "GET", Path: "/api/users?page=2", Status: status}
	}
	return entries
}

func BenchmarkReport(b *testing.B) {
	for _, n := range []int{100, 1000} {
		b.Run(fmt.Sprintf("entries=%d", n), func(b *testing.B) {
			entries := sample(n)
			for b.Loop() {
				_ = accesslog.Report(entries)
			}
		})
	}
}

func BenchmarkObserve(b *testing.B) {
	var m accesslog.Metrics
	paths := make([]string, 64)
	for i := range paths {
		paths[i] = fmt.Sprintf("/api/items/%d", i)
	}
	b.RunParallel(func(pb *testing.PB) {
		for i := 0; pb.Next(); i++ {
			m.Observe("GET", paths[i%len(paths)])
		}
	})
}
