package orders

import (
	"fmt"
	"log"
	"net/http"
)

// ANCHOR: handler

// Handler 是 HTTP 邊界：回傳待處理的訂單數。
type Handler struct {
	store *Store
}

// NewHandler 回傳使用 store 的 Handler。
func NewHandler(store *Store) *Handler {
	return &Handler{store: store}
}

// ServeHTTP 實作 http.Handler。
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	n, err := h.store.PendingCount(r.Context()) // handler 老實地把請求的 context 交下去了
	if err != nil {
		log.Printf("pending orders: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	fmt.Fprintln(w, n)
}

// ANCHOR_END: handler
