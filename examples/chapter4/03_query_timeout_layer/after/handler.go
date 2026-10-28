package orders

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"
)

// Handler 是 HTTP 邊界：回傳待處理的訂單數，並決定整個請求的 deadline。
type Handler struct {
	store   *Store
	timeout time.Duration
}

// NewHandler 回傳使用 store 的 Handler；timeout 是每個請求的 deadline。
func NewHandler(store *Store, timeout time.Duration) *Handler {
	return &Handler{store: store, timeout: timeout}
}

// ANCHOR: handler

// ServeHTTP 實作 http.Handler。
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// r.Context() 在使用者斷線時取消；邊界在它上面再加 deadline，一起往下傳。
	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()

	n, err := h.store.PendingCount(ctx)
	switch {
	case errors.Is(err, context.Canceled):
		// 使用者已經離開，沒有人在等回應。
	case errors.Is(err, context.DeadlineExceeded):
		http.Error(w, "query timed out", http.StatusGatewayTimeout)
	case err != nil:
		log.Printf("pending orders: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
	default:
		fmt.Fprintln(w, n)
	}
}

// ANCHOR_END: handler
