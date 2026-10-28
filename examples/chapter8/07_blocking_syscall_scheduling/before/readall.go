// Package sourceread 讀取一批資料來源並加總讀到的位元組數（before：每個來源一個 goroutine，沒有上限）。
package sourceread

import (
	"context"
	"sync/atomic"

	"golang.org/x/sync/errgroup"
)

// ANCHOR: readall

// ReadAll 對 sources 的每一個來源呼叫 read，回傳讀到的位元組總數與第一個錯誤。
func ReadAll[S any](ctx context.Context, sources []S, read func(context.Context, S) (int, error)) (int, error) {
	var total atomic.Int64
	g, ctx := errgroup.WithContext(ctx)
	for _, src := range sources {
		g.Go(func() error {
			n, err := read(ctx, src)
			total.Add(int64(n))
			return err
		})
	}
	err := g.Wait()
	return int(total.Load()), err
}

// ANCHOR_END: readall
