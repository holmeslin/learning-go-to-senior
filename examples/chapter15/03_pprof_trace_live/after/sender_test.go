package reminder_test

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	reminder "github.com/holmeslin/learning-go-to-senior/examples/chapter15/03_pprof_trace_live/after"
)

// supplierURL 只用來組請求：測試的 RoundTripper 不連網路。
const supplierURL = "http://supplier.invalid/push"

// roundTripFunc 讓一個函式充當 http.RoundTripper：測試不連網路，供應商的行為由它決定。
type roundTripFunc func(*http.Request) (*http.Response, error)

// RoundTrip 呼叫 f。
func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// status 回傳只有狀態碼的回應。
func status(r *http.Request, code int) *http.Response {
	return &http.Response{StatusCode: code, Body: http.NoBody, Request: r}
}

// pushes 產生 n 則提醒。
func pushes(n int) []reminder.Push {
	out := make([]reminder.Push, n)
	for i := range out {
		out[i] = reminder.Push{Key: "event-1/attendee-" + strconv.Itoa(i)}
	}
	return out
}

// TestSendAllDelivers 檢查供應商正常回應時，每一則提醒都送到、沒有錯誤。
func TestSendAllDelivers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var calls atomic.Int32
		s := reminder.Sender{
			Client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				return status(r, http.StatusNoContent), nil
			})},
			URL: supplierURL,
		}

		if err := s.SendAll(t.Context(), pushes(5)); err != nil {
			t.Fatalf("SendAll() error = %v, want nil", err)
		}
		if got, want := calls.Load(), int32(5); got != want {
			t.Errorf("supplier got %d requests, want %d", got, want)
		}
	})
}

// ANCHOR: timeout

// stuckSupplier 模擬卡住的供應商：連線建好、請求送出，就是不回應，直到呼叫端放棄。
func stuckSupplier(r *http.Request) (*http.Response, error) {
	<-r.Context().Done()
	return nil, r.Context().Err()
}

// TestSendAllTimesOut 傳進沒有 deadline 的 ctx：逾時要由 SendAll 自己負責。
// 時間是 bubble 的假時間，2 秒瞬間走完。
func TestSendAllTimesOut(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := reminder.Sender{Client: &http.Client{Transport: roundTripFunc(stuckSupplier)}, URL: supplierURL}

		start := time.Now()
		err := s.SendAll(context.Background(), pushes(1))

		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("SendAll() error = %v, want context.DeadlineExceeded", err)
		}
		if took, want := time.Since(start), 2*time.Second; took != want {
			t.Errorf("SendAll() took %v, want %v", took, want)
		}
	})
}

// TestSendAllLeavesNoGoroutine 讓供應商整個卡住、提醒比上限多：每一輪等 2 秒，20 則分 3 輪，6 秒後全部放棄。
// SendAll 回傳時不能還有 goroutine 卡著：bubble 結束時若還有，synctest.Test 會 panic。
func TestSendAllLeavesNoGoroutine(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := reminder.Sender{Client: &http.Client{Transport: roundTripFunc(stuckSupplier)}, URL: supplierURL}

		start := time.Now()
		err := s.SendAll(context.Background(), pushes(20))

		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("SendAll() error = %v, want context.DeadlineExceeded", err)
		}
		if took, want := time.Since(start), 6*time.Second; took != want {
			t.Errorf("SendAll() took %v, want %v", took, want)
		}
	})
}

// ANCHOR_END: timeout

// gauge 記錄同時進行的請求數與它的最大值。
type gauge struct {
	mu           sync.Mutex
	now, maximum int
}

func (g *gauge) add(n int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.now += n
	g.maximum = max(g.maximum, g.now)
}

func (g *gauge) peak() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.maximum
}

// TestSendAllRespectsLimit 讓供應商正常、每個請求花 100 毫秒（假時間）。30 則提醒、上限 8：
// 同時進行的請求最多 8 個，分 4 輪送完。
func TestSendAllRespectsLimit(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var inFlight gauge
		s := reminder.Sender{Client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			inFlight.add(1)
			defer inFlight.add(-1)
			time.Sleep(100 * time.Millisecond)
			return status(r, http.StatusNoContent), nil
		})}, URL: supplierURL}

		start := time.Now()
		if err := s.SendAll(t.Context(), pushes(30)); err != nil {
			t.Fatalf("SendAll() error = %v, want nil", err)
		}

		if got, want := inFlight.peak(), 8; got != want {
			t.Errorf("peak concurrent requests = %d, want %d", got, want)
		}
		if took, want := time.Since(start), 400*time.Millisecond; took != want {
			t.Errorf("SendAll() took %v, want %v", took, want)
		}
	})
}

// TestSendAllReportsRejection 檢查供應商拒絕時，錯誤指出是哪一則提醒。
func TestSendAllReportsRejection(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := reminder.Sender{
			Client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return status(r, http.StatusServiceUnavailable), nil
			})},
			URL: supplierURL,
		}

		err := s.SendAll(t.Context(), pushes(1))
		if err == nil || !strings.Contains(err.Error(), "event-1/attendee-0") {
			t.Errorf("SendAll() error = %v, want an error naming the push", err)
		}
	})
}
