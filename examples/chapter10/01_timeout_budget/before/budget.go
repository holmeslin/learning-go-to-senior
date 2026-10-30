// Package budget 是運費試算服務：handler 先查庫存、再查運費，
// 每一層各自設一個固定的逾時。
package budget

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// ANCHOR: layers

// Backend 是一次下游呼叫。收到的 ctx 結束，它就該回來。
type Backend func(ctx context.Context) error

// Client 包住一個下游，逾時是它自己的固定值。
// 它不收呼叫端的 ctx，就像 http.Client 的 Timeout 搭配沒帶 context 的 request。
type Client struct {
	Timeout time.Duration // 這個下游最多等多久
	Backend Backend       // 實際的下游呼叫
}

// Do 呼叫下游，最多等 Timeout。
func (c Client) Do() error {
	ctx, cancel := context.WithTimeout(context.Background(), c.Timeout)
	defer cancel()
	return c.Backend(ctx)
}

// Service 依序查庫存與運費。
type Service struct {
	Stock   Client // 庫存
	Carrier Client // 運費
}

// Quote 依序呼叫兩個下游，任何一個失敗就回傳錯誤。
func (s Service) Quote() error {
	if err := s.Stock.Do(); err != nil {
		return err
	}
	return s.Carrier.Do()
}

// Handler 回應運費試算，自己也有一個固定的逾時。
type Handler struct {
	Service Service       // 試算用的 service
	Timeout time.Duration // 整個請求最多等多久
}

// ServeHTTP 在請求的 ctx 或 Timeout 先到時回 504，不再等 Service。
func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.Timeout)
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- h.Service.Quote() }()

	select {
	case err := <-done:
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		fmt.Fprint(w, "ok")
	case <-ctx.Done():
		http.Error(w, "timeout", http.StatusGatewayTimeout)
	}
}

// ANCHOR_END: layers
