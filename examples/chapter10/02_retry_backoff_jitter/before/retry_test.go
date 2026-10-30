package rates_test

import (
	"context"
	"maps"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	rates "github.com/holmeslin/learning-go-to-senior/examples/chapter10/02_retry_backoff_jitter/before"
)

// ANCHOR: fake

const target = "http://rates.test/usd"

// downstream 是假的下游：直接當 http.Client 的 Transport，不開 port，
// 並記下每次呼叫距離 start 的假時間。前 failFirst 次呼叫回 503，之後回 200。
type downstream struct {
	mu        sync.Mutex
	start     time.Time
	at        []time.Duration
	failFirst int
}

func (d *downstream) RoundTrip(r *http.Request) (*http.Response, error) {
	d.mu.Lock()
	call := len(d.at)
	d.at = append(d.at, time.Since(d.start))
	d.mu.Unlock()
	rec := httptest.NewRecorder()
	if call < d.failFirst {
		rec.WriteHeader(http.StatusServiceUnavailable)
	}
	return rec.Result(), nil
}

// perInstant 數出每個時間點各有幾次呼叫。
func perInstant(at []time.Duration) map[time.Duration]int {
	counts := map[time.Duration]int{}
	for _, t := range at {
		counts[t]++
	}
	return counts
}

// ANCHOR_END: fake

// ANCHOR: immediate

func TestImmediateRetryHitsDownstreamAtOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := &downstream{start: time.Now(), failFirst: 50}
		var wg sync.WaitGroup
		for range 10 {
			wg.Go(func() { _, _ = rates.Fetch(context.Background(), &http.Client{Transport: d}, target, 0) })
		}
		wg.Wait()

		// 立即重試不會讓時間往前走：下游恢復之前的 50 次失敗，全部擠在同一個時間點。
		if len(d.at) < 50 {
			t.Fatalf("%d calls, want at least the 50 that fail", len(d.at))
		}
		if got := perInstant(d.at); len(got) != 1 {
			t.Errorf("calls spread over %v, want all at the same instant", got)
		}
	})
}

// ANCHOR_END: immediate

// ANCHOR: lockstep

func TestBackoffWithoutJitterRetriesInLockstep(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := &downstream{start: time.Now(), failFirst: 20}
		var wg sync.WaitGroup
		for range 10 {
			wg.Go(func() { _, _ = rates.Fetch(context.Background(), &http.Client{Transport: d}, target, time.Second) })
		}
		wg.Wait()

		// 退避 1s、2s、4s…，但 10 個 client 算出同一個時間：每一輪 10 次呼叫同時抵達。
		want := map[time.Duration]int{0: 10, time.Second: 10, 3 * time.Second: 10}
		if got := perInstant(d.at); !maps.Equal(got, want) {
			t.Errorf("calls per instant = %v, want %v", got, want)
		}
	})
}

// ANCHOR_END: lockstep
