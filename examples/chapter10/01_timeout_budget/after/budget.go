// Package budget 是運費試算服務：整條呼叫鏈共用邊界設下的 deadline，
// 每一步只拿到扣掉後續工作之後剩下的時間。
package budget

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// ANCHOR: within

// ErrNoBudget 表示剩下的時間不夠做這一步，下游沒有被呼叫。
// 呼叫端可以用 errors.Is 把它和下游自己的逾時分開。
var ErrNoBudget = errors.New("budget: not enough time left")

// within 把 ctx 的 deadline 往前挪 reserve，留給呼叫端自己的後續工作，
// 再交給 fn。剩下的時間不到 need 就直接回 ErrNoBudget，不呼叫 fn。
// ctx 沒有 deadline 時原樣交給 fn：deadline 由最外層的邊界負責設。
func within(ctx context.Context, reserve, need time.Duration, fn Backend) error {
	deadline, ok := ctx.Deadline()
	if !ok {
		return fn(ctx)
	}
	if left := time.Until(deadline) - reserve; left < need {
		return fmt.Errorf("%w: %v left after reserving %v, need %v", ErrNoBudget, left, reserve, need)
	}
	ctx, cancel := context.WithDeadline(ctx, deadline.Add(-reserve))
	defer cancel()
	return fn(ctx)
}

// ANCHOR_END: within

// ANCHOR: service

// Backend 是一次下游呼叫。收到的 ctx 結束，它就該回來。
type Backend func(ctx context.Context) error

const (
	stockNeed   = 500 * time.Millisecond // 查庫存至少要這麼久才有意義
	carrierNeed = time.Second            // 查運費至少要這麼久才有意義
	respond     = 200 * time.Millisecond // 組出並寫完回應要留的時間
)

// Service 依序查庫存與運費，兩步共用同一份預算。
type Service struct {
	Stock   Backend // 庫存
	Carrier Backend // 運費
}

// Quote 依序呼叫兩個下游。庫存之後要先留下運費與回應需要的時間，
// 所以庫存再慢也吃不掉後面的預算。ctx 的 deadline 不夠時回傳 ErrNoBudget。
func (s Service) Quote(ctx context.Context) error {
	if err := within(ctx, carrierNeed+respond, stockNeed, s.Stock); err != nil {
		return fmt.Errorf("stock: %w", err)
	}
	if err := within(ctx, respond, carrierNeed, s.Carrier); err != nil {
		return fmt.Errorf("carrier: %w", err)
	}
	return nil
}

// ANCHOR_END: service

// ANCHOR: handler

// Handler 回應運費試算。
type Handler struct {
	Service Service       // 試算用的 service
	Budget  time.Duration // 呼叫端願意等的總時間
}

// ServeHTTP 在請求的 ctx 上再加 Budget，兩者取較早的，作為整條呼叫鏈的 deadline。
func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.Budget)
	defer cancel()

	err := h.Service.Quote(ctx)
	switch {
	case err == nil:
		fmt.Fprint(w, "ok")
	case errors.Is(err, context.Canceled):
		// 呼叫端已經走了，沒有人在等回應。
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, ErrNoBudget):
		http.Error(w, "timeout", http.StatusGatewayTimeout)
	default:
		http.Error(w, err.Error(), http.StatusBadGateway)
	}
}

// ANCHOR_END: handler
