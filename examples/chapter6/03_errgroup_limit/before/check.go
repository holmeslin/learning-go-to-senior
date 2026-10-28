// Package addrcheck 對一批地址逐筆呼叫下游的地址驗證服務（before：每筆一個 goroutine，沒有上限）。
package addrcheck

import (
	"context"

	"golang.org/x/sync/errgroup"
)

// ANCHOR: checkall

// CheckAll 對 addrs 的每一筆呼叫 check，回傳第一個錯誤。
func CheckAll(ctx context.Context, addrs []string, check func(context.Context, string) error) error {
	g, ctx := errgroup.WithContext(ctx)
	for _, addr := range addrs {
		g.Go(func() error { return check(ctx, addr) })
	}
	return g.Wait()
}

// ANCHOR_END: checkall
