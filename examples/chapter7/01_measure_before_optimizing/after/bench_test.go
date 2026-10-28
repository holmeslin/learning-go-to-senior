package logstat_test

import (
	"fmt"
	"testing"

	logstat "github.com/holmeslin/learning-go-to-senior/examples/chapter7/01_measure_before_optimizing/after"
)

// logLines 產生 n 行固定內容的日誌，路由與狀態碼的比例固定，不依賴亂數。
func logLines(n int) []string {
	statuses := []int{200, 200, 200, 404, 500}
	lines := make([]string, n)
	for i := range lines {
		if i%10 == 9 {
			lines[i] = lines[i-1] // 重送：同一個請求編號再出現一次
			continue
		}
		var path string
		switch i % 5 {
		case 0:
			path = fmt.Sprintf("/users/%d", i)
		case 1:
			path = fmt.Sprintf("/users/%d/orders/%d", i, i/2)
		case 2:
			path = fmt.Sprintf("/orders/%d", i)
		case 3:
			path = "/health"
		default:
			path = "/static/app.js"
		}
		lines[i] = fmt.Sprintf("r%d GET %s %d %d", i, path, statuses[i%len(statuses)], i%97)
	}
	return lines
}

// BenchmarkSummarize 量一批 20000 行日誌的彙總；準備工作在迴圈外。
// 名稱與 before 的 impl=simple 相同，benchstat 才能把兩邊的結果並排。
func BenchmarkSummarize(b *testing.B) {
	lines := logLines(20000)
	b.Run("lines=20000", func(b *testing.B) {
		b.Run("impl=simple", func(b *testing.B) {
			for b.Loop() {
				logstat.Summarize(lines)
			}
		})
	})
}
