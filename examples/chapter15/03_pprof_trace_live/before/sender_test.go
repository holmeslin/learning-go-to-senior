package reminder_test

import (
	"net/http"
	"strconv"
	"sync/atomic"
	"testing"
	"testing/synctest"

	reminder "github.com/holmeslin/learning-go-to-senior/examples/chapter15/03_pprof_trace_live/before"
)

// roundTripFunc 讓一個函式充當 http.RoundTripper：測試不連網路，供應商的行為由它決定。
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// 供應商正常回應的路徑：before 只測這一條，而且會通過。
// SendAll 不等結果，所以測試用 synctest.Wait 等所有 goroutine 都做完。
func TestSendAllDelivers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var calls atomic.Int32
		s := reminder.Sender{
			Client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				return &http.Response{StatusCode: http.StatusNoContent, Body: http.NoBody, Request: r}, nil
			})},
			URL: "http://supplier.invalid/push",
		}
		pushes := make([]reminder.Push, 5)
		for i := range pushes {
			pushes[i] = reminder.Push{Key: "event-1/attendee-" + strconv.Itoa(i)}
		}

		if err := s.SendAll(t.Context(), pushes); err != nil {
			t.Fatalf("SendAll() error = %v, want nil", err)
		}
		synctest.Wait()

		if got, want := calls.Load(), int32(5); got != want {
			t.Errorf("supplier got %d requests, want %d", got, want)
		}
	})
}
