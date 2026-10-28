package quote_test

import (
	"testing"
	"testing/synctest"
	"time"

	quote "github.com/holmeslin/learning-go-to-senior/examples/chapter6/09_goroutine_leak/after"
)

// ANCHOR: leak

// 兩個供應商的延遲。睡的是 bubble 的假時間，測試瞬間跑完。
const (
	fastDelay = time.Second
	slowDelay = time.Hour
)

// Fastest 回傳後，bubble 結束前所有 goroutine 都要走完；
// 輸家的送出若卡住，synctest.Test 會 panic。
func TestFastestLeavesNoGoroutine(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		fast := func() string { time.Sleep(fastDelay); return "fast" }
		slow := func() string { time.Sleep(slowDelay); return "slow" }

		start := time.Now()
		got := quote.Fastest([]quote.Source{slow, fast})
		if want := "fast"; got != want {
			t.Errorf("Fastest() = %q, want %q", got, want)
		}
		if elapsed, want := time.Since(start), fastDelay; elapsed != want {
			t.Errorf("Fastest() took %v, want %v", elapsed, want)
		}

		// 時間只在 bubble 的主 goroutine 還活著時前進；
		// 睡過慢的供應商的延遲，輸家才有機會跑到送出那一行。
		time.Sleep(slowDelay)
	})
}

// ANCHOR_END: leak

func TestFastestSingleSource(t *testing.T) {
	got := quote.Fastest([]quote.Source{func() string { return "only" }})
	if want := "only"; got != want {
		t.Errorf("Fastest() = %q, want %q", got, want)
	}
}
