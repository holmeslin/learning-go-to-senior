package quote_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/synctest"
	"time"

	quote "github.com/holmeslin/learning-go-to-senior/examples/chapter10/05_circuit_breaker/before"
)

// ANCHOR: test

func TestEveryRequestWaitsForTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		calls := 0
		carrierDown := func(ctx context.Context) (int, error) {
			calls++
			<-ctx.Done() // 下游掛了：不回應，只等到逾時
			return 0, ctx.Err()
		}
		h := quote.Handler{Downstream: carrierDown, Timeout: 3 * time.Second}

		start := time.Now()
		for range 10 {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/quote", nil))
			if rec.Code != http.StatusGatewayTimeout {
				t.Fatalf("GET /quote = %d, want 504", rec.Code)
			}
		}

		if calls != 10 {
			t.Errorf("downstream calls = %d, want 10 (every request reaches the dead carrier)", calls)
		}
		if got := time.Since(start); got != 30*time.Second {
			t.Errorf("10 requests took %v in the bubble, want 30s (3s each)", got)
		}
	})
}

// ANCHOR_END: test
