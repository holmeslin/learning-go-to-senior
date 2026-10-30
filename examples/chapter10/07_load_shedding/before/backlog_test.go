package loadshed_test

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	loadshed "github.com/holmeslin/learning-go-to-senior/examples/chapter10/07_load_shedding/before"
)

// ANCHOR: backlog

func TestBacklogMakesEveryRequestSlow(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var worked atomic.Int32
		h := loadshed.NewHandler(2, func() { // 每個請求做 1 秒，同時 2 個：每秒做完 2 個
			worked.Add(1)
			time.Sleep(time.Second)
		})

		var (
			mu        sync.Mutex
			wg        sync.WaitGroup
			latencies []time.Duration
			codes     = map[int]int{}
		)
		for range 10 { // 每秒進來 4 個，比處理的速度多一倍，持續 10 秒
			for range 4 {
				wg.Go(func() {
					start := time.Now()
					rec := httptest.NewRecorder()
					h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
					mu.Lock()
					defer mu.Unlock()
					latencies = append(latencies, time.Since(start))
					codes[rec.Code]++
				})
			}
			time.Sleep(time.Second)
		}
		wg.Wait()

		slices.Sort(latencies)
		if codes[http.StatusOK] != 40 {
			t.Errorf("responses = %v, want 40 x 200 (nothing is ever refused)", codes)
		}
		// 假時間是確定的，延遲可以精確斷言：排在後面的請求一個比一個等得久。
		if latencies[0] != time.Second || latencies[39] != 11*time.Second {
			t.Errorf("latency from %v to %v, want 1s to 11s", latencies[0], latencies[39])
		}
		// 呼叫端只願意等 3 秒：前 10 個趕得上，之後每一個都來不及。
		onTime := 0
		for _, d := range latencies {
			if d <= 3*time.Second {
				onTime++
			}
		}
		if onTime != 10 {
			t.Errorf("answered within 3s = %d, want 10 of 40", onTime)
		}
		if got := worked.Load(); got != 40 {
			t.Errorf("jobs run = %d, want 40 (the server still works for callers that gave up)", got)
		}
	})
}

// ANCHOR_END: backlog
