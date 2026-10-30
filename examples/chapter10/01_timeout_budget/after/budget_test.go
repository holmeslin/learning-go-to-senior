package budget_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	budget "github.com/holmeslin/learning-go-to-senior/examples/chapter10/01_timeout_budget/after"
)

// ANCHOR: serve

// serve 對 handler 發一個 GET，回傳回應與 handler 花的假時間。
func serve(ctx context.Context, h budget.Handler) (*httptest.ResponseRecorder, time.Duration) {
	start := time.Now()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequestWithContext(ctx, http.MethodGet, "/quote", nil))
	return w, time.Since(start)
}

// newHandler 組出預算 3 秒的 handler，兩個下游各自要花 stock 與 carrier 這麼久。
func newHandler(rec *recorder, stock, carrier time.Duration) budget.Handler {
	return budget.Handler{
		Budget:  3 * time.Second,
		Service: budget.Service{Stock: rec.backend("stock", stock), Carrier: rec.backend("carrier", carrier)},
	}
}

// checkLog 比對 recorder 記下的下游呼叫與預期是否一致。
func checkLog(t *testing.T, rec *recorder, want ...string) {
	t.Helper()
	if got := rec.entries(); !slices.Equal(got, want) {
		t.Errorf("downstream calls:\n got %q\nwant %q", got, want)
	}
}

// ANCHOR_END: serve

// ANCHOR: within

func TestEachStepGetsWhatIsLeft(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		rec := newRecorder()
		w, took := serve(context.Background(), newHandler(rec, 500*time.Millisecond, time.Second))

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want 200", w.Code)
		}
		// 庫存之後要留 1 秒給運費、200 毫秒給回應；運費只扣回應的 200 毫秒。
		checkLog(t, rec,
			"0s stock start, deadline in 1.8s",
			"500ms stock done",
			"500ms carrier start, deadline in 2.3s",
			"1.5s carrier done",
		)
		if took != 1500*time.Millisecond {
			t.Errorf("took %v, want 1.5s", took)
		}
	})
}

// ANCHOR_END: within

// ANCHOR: slow

func TestSlowDownstreamGivesUpInsideBudget(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		rec := newRecorder()
		w, took := serve(context.Background(), newHandler(rec, 4*time.Second, 4*time.Second))

		if w.Code != http.StatusGatewayTimeout {
			t.Errorf("status = %d, want 504", w.Code)
		}
		checkLog(t, rec,
			"0s stock start, deadline in 1.8s",
			"1.8s stock aborted", // 沒有 carrier：留給它的時間不能被庫存吃掉
		)
		if took != 1800*time.Millisecond {
			t.Errorf("took %v, want 1.8s, inside the 3s budget", took)
		}
	})
}

// ANCHOR_END: slow

// ANCHOR: nobudget

func TestNotEnoughBudgetSkipsDownstream(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		rec := newRecorder()
		// 呼叫端只剩 1 秒：庫存之後要留的 1.2 秒已經不夠，連庫存都不必呼叫。
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		w, took := serve(ctx, newHandler(rec, 500*time.Millisecond, time.Second))

		if w.Code != http.StatusGatewayTimeout {
			t.Errorf("status = %d, want 504", w.Code)
		}
		checkLog(t, rec)
		if took != 0 {
			t.Errorf("took %v, want 0: the error comes before any waiting", took)
		}
	})
}

// ANCHOR_END: nobudget

func TestCallerLeavingStopsTheChain(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		rec := newRecorder()
		ctx, cancel := context.WithCancel(context.Background())
		time.AfterFunc(time.Second, cancel) // 呼叫端 1 秒後斷線
		w, took := serve(ctx, newHandler(rec, 4*time.Second, 4*time.Second))

		if w.Body.Len() != 0 {
			t.Errorf("body = %q, want nothing written to a caller that left", w.Body)
		}
		checkLog(t, rec,
			"0s stock start, deadline in 1.8s",
			"1s stock aborted",
		)
		if took != time.Second {
			t.Errorf("took %v, want 1s", took)
		}
	})
}
