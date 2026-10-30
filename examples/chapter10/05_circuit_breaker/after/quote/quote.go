// Package quote 是較好的做法：下游掛了，斷路器打開之後立刻回 503。
package quote

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter10/05_circuit_breaker/after/breaker"
)

// Downstream 向承運商查運費；成功回傳運費。
type Downstream func(ctx context.Context) (fee int, err error)

// ANCHOR: handler

// Handler 處理 GET /quote：經過斷路器把請求轉給承運商。
type Handler struct {
	Downstream Downstream
	Breaker    *breaker.Breaker
	Timeout    time.Duration // 每次呼叫承運商最多等多久
	RetryAfter time.Duration // 斷路器擋下請求時，建議呼叫端多久之後再來
}

// ServeHTTP 實作 http.Handler。
func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var fee int
	err := h.Breaker.Do(func() error {
		ctx, cancel := context.WithTimeout(r.Context(), h.Timeout)
		defer cancel()
		var err error
		fee, err = h.Downstream(ctx)
		return err
	})

	switch {
	case errors.Is(err, breaker.ErrOpen):
		w.Header().Set("Retry-After", strconv.Itoa(int(h.RetryAfter.Seconds())))
		http.Error(w, "carrier unavailable", http.StatusServiceUnavailable)
	case errors.Is(err, context.DeadlineExceeded):
		http.Error(w, "carrier timeout", http.StatusGatewayTimeout)
	case err != nil:
		http.Error(w, "carrier error", http.StatusBadGateway)
	default:
		fmt.Fprint(w, fee)
	}
}

// ANCHOR_END: handler
