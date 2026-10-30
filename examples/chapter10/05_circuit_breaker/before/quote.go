// Package quote 是常見寫法：運費報價服務把請求轉給承運商，下游掛了也照打。
package quote

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// Downstream 向承運商查運費；成功回傳運費。
type Downstream func(ctx context.Context) (fee int, err error)

// ANCHOR: handler

// Handler 處理 GET /quote：把請求轉給承運商，最多等 Timeout。
type Handler struct {
	Downstream Downstream    // 向承運商查運費
	Timeout    time.Duration // 每次呼叫承運商最多等多久
}

// ServeHTTP 實作 http.Handler。
func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.Timeout)
	defer cancel()

	fee, err := h.Downstream(ctx)
	if errors.Is(err, context.DeadlineExceeded) {
		http.Error(w, "carrier timeout", http.StatusGatewayTimeout)
		return
	}
	if err != nil {
		http.Error(w, "carrier error", http.StatusBadGateway)
		return
	}
	fmt.Fprint(w, fee)
}

// ANCHOR_END: handler
