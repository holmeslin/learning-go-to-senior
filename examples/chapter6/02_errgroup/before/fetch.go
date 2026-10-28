// Package fetch 同時抓取多個來源。這是 before：用 WaitGroup 與 mutex 手刻「回傳第一個錯誤」。
package fetch

import (
	"context"
	"sync"
)

// Fetch 抓取單一來源 src。ctx 取消時，實作應盡快回傳。
type Fetch func(ctx context.Context, src string) error

// ANCHOR: fetchall

// FetchAll 同時抓取 srcs 裡的每個來源，等全部結束後回傳錯誤。
func FetchAll(ctx context.Context, srcs []string, fetch Fetch) error {
	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		firstErr error
	)
	for _, src := range srcs {
		wg.Go(func() {
			if err := fetch(ctx, src); err != nil {
				mu.Lock()
				firstErr = err
				mu.Unlock()
			}
		})
	}
	wg.Wait()
	return firstErr
}

// ANCHOR_END: fetchall
