package budget_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	budget "github.com/holmeslin/learning-go-to-senior/examples/chapter10/01_timeout_budget/before"
)

// ANCHOR: test

func TestWorkContinuesAfterCallerGivesUp(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		rec := newRecorder()
		h := budget.Handler{
			Timeout: 5 * time.Second,
			Service: budget.Service{
				Stock:   budget.Client{Timeout: 5 * time.Second, Backend: rec.backend("stock", 4*time.Second)},
				Carrier: budget.Client{Timeout: 5 * time.Second, Backend: rec.backend("carrier", 4*time.Second)},
			},
		}

		// 呼叫端只願意等 3 秒。
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequestWithContext(ctx, http.MethodGet, "/quote", nil))
		gaveUp := time.Since(rec.start)

		synctest.Sleep(10 * time.Second) // 讓還在做的下游走完

		if gaveUp != 3*time.Second {
			t.Errorf("handler returned after %v, want 3s (the caller's patience)", gaveUp)
		}
		want := []string{
			"0s stock start, deadline in 5s",
			"4s stock done",
			"4s carrier start, deadline in 5s", // 呼叫端已經走了 1 秒
			"8s carrier done",
		}
		if got := rec.entries(); !slices.Equal(got, want) {
			t.Errorf("downstream calls:\n got %q\nwant %q", got, want)
		}
	})
}

// ANCHOR_END: test
