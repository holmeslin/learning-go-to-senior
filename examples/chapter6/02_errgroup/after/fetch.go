// Package fetch 同時抓取多個來源。這是 after：用 errgroup 回傳第一個錯誤並取消其他來源。
package fetch

import (
	"context"

	"golang.org/x/sync/errgroup"
)

// Fetch 抓取單一來源 src。ctx 取消時，實作應盡快回傳。
type Fetch func(ctx context.Context, src string) error

// ANCHOR: fetchall

// FetchAll 同時抓取 srcs 裡的每個來源。第一個失敗的來源會取消其他來源，回傳的就是它的錯誤。
func FetchAll(ctx context.Context, srcs []string, fetch Fetch) error {
	g, ctx := errgroup.WithContext(ctx)
	for _, src := range srcs {
		g.Go(func() error { return fetch(ctx, src) })
	}
	return g.Wait()
}

// ANCHOR_END: fetchall
