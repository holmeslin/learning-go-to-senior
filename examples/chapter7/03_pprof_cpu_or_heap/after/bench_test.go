package logstats_test

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	logstats "github.com/holmeslin/learning-go-to-senior/examples/chapter7/03_pprof_cpu_or_heap/after"
)

// ANCHOR: bench

func BenchmarkSummarize(b *testing.B) {
	lines := sampleLines(2000)
	for b.Loop() {
		if _, err := logstats.Summarize(lines); err != nil {
			b.Fatal(err)
		}
	}
}

// ANCHOR_END: bench

// ANCHOR: lines

// sampleLines 產生 n 行固定內容的 access log：五種路徑輪流出現，延遲與狀態碼由 i 決定。
func sampleLines(n int) []string {
	paths := []string{"/api/users/{id}", "/api/orders/{id}/items", "/api/search", "/healthz", "/static/app.js"}
	lines := make([]string, n)
	for i := range lines {
		status := 200
		if i%50 == 0 {
			status = 503
		}
		path := strings.ReplaceAll(paths[i%len(paths)], "{id}", strconv.Itoa(i%1000))
		lines[i] = fmt.Sprintf("2026-10-28T10:%02d:%02dZ GET %s %d %d", i/60%60, i%60, path, status, 5+i*7%90)
	}
	return lines
}

// ANCHOR_END: lines
