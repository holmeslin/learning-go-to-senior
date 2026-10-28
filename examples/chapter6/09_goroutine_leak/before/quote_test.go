package quote_test

import (
	"testing"

	quote "github.com/holmeslin/learning-go-to-senior/examples/chapter6/09_goroutine_leak/before"
)

// 只測單一供應商：兩個以上就一定有輸家，輸家會卡在送出，
// repo 裡不放會讓測試結束時留下卡住 goroutine 的測試。
func TestFastestSingleSource(t *testing.T) {
	got := quote.Fastest([]quote.Source{func() string { return "only" }})
	if want := "only"; got != want {
		t.Errorf("Fastest() = %q, want %q", got, want)
	}
}
