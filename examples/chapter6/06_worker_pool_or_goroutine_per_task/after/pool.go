package batch

import (
	"context"
	"io"

	"golang.org/x/sync/errgroup"
)

// ANCHOR: consume

// Consume 啟動 workers 個 worker，每個 worker 先用 dial 建立自己的連線，
// 再從 jobs 取工作交給 handle，直到 jobs 關閉、ctx 結束或有人出錯。
// 連線在 worker 結束時關閉。第一個錯誤會取消其他 worker，並成為回傳的錯誤。
func Consume[C io.Closer, J any](
	ctx context.Context,
	workers int,
	dial func() (C, error),
	jobs <-chan J,
	handle func(ctx context.Context, conn C, job J) error,
) error {
	g, ctx := errgroup.WithContext(ctx)
	for range workers {
		g.Go(func() error {
			conn, err := dial() // 每個 worker 建立一次，不是每個工作一次
			if err != nil {
				return err
			}
			defer conn.Close()
			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case job, ok := <-jobs:
					if !ok {
						return nil
					}
					if err := handle(ctx, conn, job); err != nil {
						return err
					}
				}
			}
		})
	}
	return g.Wait()
}

// ANCHOR_END: consume
