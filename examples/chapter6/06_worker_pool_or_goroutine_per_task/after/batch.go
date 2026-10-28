// Package batch 批次抓取一批頁面，同一時間最多 limit 個。
// after 版本每個任務一個 goroutine，上限交給 errgroup。
package batch

import (
	"context"
	"errors"

	"golang.org/x/sync/errgroup"
)

// Fetch 取得 id 對應的頁面標題。ctx 結束時應盡快回傳。
type Fetch func(ctx context.Context, id int) (string, error)

// ANCHOR: errgroup

// FetchAll 對 ids 裡的每個 id 呼叫 fetch，同一時間最多 limit 個，
// 回傳與 ids 同順序的結果。第一個錯誤會取消傳給 fetch 的 ctx，並成為回傳的錯誤。
// limit 小於 1 時回傳錯誤。
func FetchAll(ctx context.Context, ids []int, limit int, fetch Fetch) ([]string, error) {
	if limit < 1 {
		return nil, errors.New("limit must be at least 1")
	}
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(limit)

	titles := make([]string, len(ids))
	for i, id := range ids {
		g.Go(func() error {
			title, err := fetch(ctx, id)
			titles[i] = title // 每個 goroutine 只寫自己的索引
			return err
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return titles, nil
}

// ANCHOR_END: errgroup
