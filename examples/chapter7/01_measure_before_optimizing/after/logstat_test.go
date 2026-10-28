package logstat_test

import (
	"fmt"
	"slices"
	"testing"

	logstat "github.com/holmeslin/learning-go-to-senior/examples/chapter7/01_measure_before_optimizing/after"
	before "github.com/holmeslin/learning-go-to-senior/examples/chapter7/01_measure_before_optimizing/before"
)

// ANCHOR: example

func ExampleSummarize() {
	lines := []string{
		"r1 GET /users/42 200 12",
		"r2 GET /users/7 500 30",
		"r3 POST /orders/9 201 8",
		"r4 GET /users/7/orders/3 200 5",
		"r2 GET /users/7 500 30", // 重送：只算第一次
		"broken line",
	}
	for _, s := range logstat.Summarize(lines) {
		fmt.Printf("%s count=%d errors=%d total_ms=%d\n", s.Route, s.Count, s.Errors, s.TotalMS)
	}
	// Output:
	// /orders/:id count=1 errors=0 total_ms=8
	// /users/:id count=2 errors=1 total_ms=42
	// /users/:id/orders/:id count=1 errors=0 total_ms=5
}

// ANCHOR_END: example

// malformed 是壞掉的行：欄位數不對、狀態碼或毫秒不是數字、位數太多、多餘的空白。
var malformed = []string{
	"", "r1", "r1 GET /x 200", "r1 GET /x 200 5 extra", "r1 GET /x abc 5", "r1 GET /x 200 -5",
	"r1 GET /x 200 1234567890", "r1  GET /x 200 5", "r1 GET /x 2_0 5", "r1 GET /x +200 5",
}

// ANCHOR: same

// sameStats 比較兩個套件各自的 RouteStat；兩個型別的欄位完全相同，可以直接轉換。
func sameStats(got []logstat.RouteStat, want []before.RouteStat) bool {
	return slices.EqualFunc(got, want, func(g logstat.RouteStat, w before.RouteStat) bool {
		return g == logstat.RouteStat(w)
	})
}

// TestSummarizeMatchesBefore 拿最佳化前的版本當對照：同樣的輸入，結果要一樣。
func TestSummarizeMatchesBefore(t *testing.T) {
	lines := append(logLines(500), malformed...)

	want := before.SummarizeSimple(lines)
	if got := logstat.Summarize(lines); !sameStats(got, want) {
		t.Errorf("Summarize() = %v, want %v", got, want)
	}
	if got := before.Summarize(lines); !slices.Equal(got, want) {
		t.Errorf("before.Summarize() = %v, want %v", got, want)
	}
}

// ANCHOR_END: same

// ANCHOR: allocs

// TestSummarizeAllocs 斷言配置次數的上限，時間不在測試裡斷言。
// 熱點改用 map，配置次數和原本差不多（每行 3 到 4 次）；上限擋住之後悄悄長大的配置。
// regexp 內部用 sync.Pool，-race 下配置次數會有出入，所以只設上限，不斷言精確次數。
func TestSummarizeAllocs(t *testing.T) {
	lines := logLines(5000)

	allocs := testing.AllocsPerRun(5, func() { logstat.Summarize(lines) })

	if perLine := allocs / float64(len(lines)); perLine >= 6 {
		t.Errorf("allocations per line = %.1f, want < 6", perLine)
	}
}

// ANCHOR_END: allocs
