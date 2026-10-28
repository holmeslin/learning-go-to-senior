// Package sourceread 讀取一批資料來源並加總讀到的位元組數，同時進行的讀取數有上限。
package sourceread

import (
	"context"
	"errors"
	"sync/atomic"

	"golang.org/x/sync/errgroup"
)

// ErrInvalidLimit 表示 limit 小於 1。
var ErrInvalidLimit = errors.New("limit must be at least 1")

// ANCHOR: readall

// ReadAll 對 sources 的每一個來源呼叫 read，回傳讀到的位元組總數與第一個錯誤。
// 同時進行的 read 不超過 limit；limit 小於 1 時回傳 ErrInvalidLimit。
func ReadAll(ctx context.Context, sources []int, limit int, read func(context.Context, int) (int, error)) (int, error) {
	if limit < 1 {
		return 0, ErrInvalidLimit
	}
	var total atomic.Int64
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(limit)
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
