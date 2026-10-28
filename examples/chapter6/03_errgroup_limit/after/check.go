// Package addrcheck 對一批地址逐筆呼叫下游的地址驗證服務，同時進行的呼叫數有上限。
package addrcheck

import (
	"context"
	"errors"

	"golang.org/x/sync/errgroup"
)

// ErrInvalidLimit 表示 limit 小於 1。
var ErrInvalidLimit = errors.New("limit must be at least 1")

// ANCHOR: checkall

// CheckAll 對 addrs 的每一筆呼叫 check，回傳第一個錯誤。
// 同時進行的 check 不超過 limit；limit 小於 1 時回傳 ErrInvalidLimit。
func CheckAll(ctx context.Context, addrs []string, limit int, check func(context.Context, string) error) error {
	if limit < 1 {
		return ErrInvalidLimit
	}
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(limit)
	for _, addr := range addrs {
		g.Go(func() error { return check(ctx, addr) })
	}
	return g.Wait()
}

// ANCHOR_END: checkall
