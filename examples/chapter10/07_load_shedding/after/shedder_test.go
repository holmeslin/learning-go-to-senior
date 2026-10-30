package loadshed_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	loadshed "github.com/holmeslin/learning-go-to-senior/examples/chapter10/07_load_shedding/after"
)

// ANCHOR: helpers

// gate 是會卡住的 handler：進來就算一個處理中的請求，open 之前不回應。
type gate struct {
	release  chan struct{}
	once     sync.Once
	inflight atomic.Int32
}

func newGate() *gate { return &gate{release: make(chan struct{})} }

// open 放行所有卡住的請求；可重複呼叫，測試失敗提早結束時用 defer 也不會讓 goroutine 卡住。
func (g *gate) open() { g.once.Do(func() { close(g.release) }) }

func (g *gate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	g.inflight.Add(1)
	defer g.inflight.Add(-1)
	<-g.release
}

// fire 同時送出 n 個請求，回應一產生就放進回傳的 channel。
func fire(h http.Handler, n int) <-chan *httptest.ResponseRecorder {
	done := make(chan *httptest.ResponseRecorder, n)
	for range n {
		go func() {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
			done <- rec
		}()
	}
	return done
}

// ANCHOR_END: helpers

// ANCHOR: shed

func TestShedsOverLimitAndKeepsReserveForCritical(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := newGate()
		defer g.open()
		s := loadshed.NewShedder(4, 2, time.Second)
		optional := s.Handler(loadshed.Optional, g, nil)
		critical := s.Handler(loadshed.Critical, g, nil)

		// 5 個 Optional 同時進來：只有 limit-reserve = 2 個能處理。
		done := fire(optional, 5)
		synctest.Wait() // 所有 goroutine 都卡住：能進的卡在 gate，被拒的已經回應
		if got := g.inflight.Load(); got != 2 {
			t.Fatalf("in flight after 5 optional = %d, want 2", got)
		}
		if got := len(done); got != 3 {
			t.Fatalf("rejected optional = %d, want 3", got)
		}
		for range 3 {
			rec := <-done
			if rec.Code != http.StatusServiceUnavailable || rec.Header().Get("Retry-After") != "1" {
				t.Errorf("rejected = %d, Retry-After %q, want 503 and \"1\"", rec.Code, rec.Header().Get("Retry-After"))
			}
		}

		// Optional 已經塞滿，保留的名額讓 Critical 還進得來；超過 limit 的第 3 個才被拒。
		done = fire(critical, 3)
		synctest.Wait()
		if got := g.inflight.Load(); got != 4 {
			t.Fatalf("in flight after 3 critical = %d, want 4 (the limit)", got)
		}
		if got := len(done); got != 1 {
			t.Fatalf("rejected critical = %d, want 1", got)
		}

		g.open()
		synctest.Wait()
		if got := g.inflight.Load(); got != 0 {
			t.Errorf("in flight after release = %d, want 0", got)
		}
		// 名額都歸還了：新的 Optional 不會被拒，直接通過已放行的 gate。
		if rec := <-fire(optional, 1); rec.Code != http.StatusOK {
			t.Errorf("optional after release = %d, want 200", rec.Code)
		}
	})
}

// ANCHOR_END: shed

// ANCHOR: degrade

func TestOptionalDegradesInsteadOfRejecting(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := newGate()
		defer g.open()
		s := loadshed.NewShedder(2, 1, time.Second)
		cached := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			io.WriteString(w, `["cached"]`)
		})
		recommend := s.Handler(loadshed.Optional, g, cached)

		done := fire(recommend, 2) // 第一個佔住 Optional 的名額，第二個降級
		synctest.Wait()
		if len(done) != 1 {
			t.Fatalf("responses = %d, want 1 (the degraded one)", len(done))
		}
		rec := <-done
		if rec.Code != http.StatusOK || rec.Body.String() != `["cached"]` {
			t.Errorf("degraded = %d %q, want 200 with cached body", rec.Code, rec.Body)
		}
		if got := rec.Header().Get("X-Degraded"); got != "overload" {
			t.Errorf("X-Degraded = %q, want \"overload\"", got)
		}
		if got := g.inflight.Load(); got != 1 {
			t.Errorf("in flight = %d, want 1 (the fallback must not run the real handler)", got)
		}
	})
}

// ANCHOR_END: degrade

// ANCHOR: waves

func TestSameTrafficRejectedQuicklyInsteadOfSlow(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		work := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(800 * time.Millisecond)
		})
		h := loadshed.NewShedder(2, 0, time.Second).Handler(loadshed.Critical, work, nil)

		var (
			mu                sync.Mutex
			wg                sync.WaitGroup
			accepted, rejects int
			slowest           time.Duration
		)
		for range 10 { // 與 before 相同：每秒 4 個請求，持續 10 秒
			for range 4 {
				wg.Go(func() {
					start := time.Now()
					rec := httptest.NewRecorder()
					h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
					mu.Lock()
					defer mu.Unlock()
					slowest = max(slowest, time.Since(start))
					if rec.Code == http.StatusOK {
						accepted++
					} else {
						rejects++
					}
				})
			}
			time.Sleep(time.Second)
		}
		wg.Wait()

		if accepted != 20 || rejects != 20 {
			t.Errorf("accepted %d, rejected %d, want 20 and 20", accepted, rejects)
		}
		if slowest != 800*time.Millisecond {
			t.Errorf("slowest response = %v, want 800ms (rejections take 0)", slowest)
		}
	})
}

// ANCHOR_END: waves

func TestRetryAfterRoundsUpToWholeSeconds(t *testing.T) {
	for _, tc := range []struct {
		wait time.Duration
		want string
	}{{time.Second, "1"}, {1500 * time.Millisecond, "2"}, {10 * time.Millisecond, "1"}} {
		h := loadshed.NewShedder(0, 0, tc.wait).Handler(loadshed.Critical, http.NotFoundHandler(), nil)
		rec := <-fire(h, 1)
		if got := rec.Header().Get("Retry-After"); got != tc.want {
			t.Errorf("Retry-After for %v = %q, want %q", tc.wait, got, tc.want)
		}
	}
}
