// Package batch 批次抓取一批頁面，同一時間最多 limit 個。
// before 版本為這個有限的批次手寫了一個 worker pool。
package batch

import (
	"context"
	"errors"
	"sync"
)

// Fetch 取得 id 對應的頁面標題。ctx 結束時應盡快回傳。
type Fetch func(ctx context.Context, id int) (string, error)

// ANCHOR: pool

// FetchAll 對 ids 裡的每個 id 呼叫 fetch，同一時間最多 limit 個，
// 回傳與 ids 同順序的結果。第一個錯誤會取消傳給 fetch 的 ctx，並成為回傳的錯誤。
// limit 小於 1 時回傳錯誤。
func FetchAll(ctx context.Context, ids []int, limit int, fetch Fetch) ([]string, error) {
	if limit < 1 {
		return nil, errors.New("limit must be at least 1")
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	type job struct{ index, id int }
	type result struct {
		index int
		title string
		err   error
	}
	jobs := make(chan job)
	results := make(chan result)

	var wg sync.WaitGroup
	for range limit {
		wg.Go(func() {
			for j := range jobs {
				title, err := fetch(ctx, j.id)
				results <- result{j.index, title, err}
			}
		})
	}
	go func() { // 送工作要在自己的 goroutine，否則和下面收結果互相等
		defer close(jobs)
		for i, id := range ids {
			jobs <- job{i, id}
		}
	}()
	go func() { // 全部 worker 結束後才能關 results
		wg.Wait()
		close(results)
	}()

	titles := make([]string, len(ids))
	var firstErr error
	for r := range results { // 出錯後仍要收完，否則 worker 會卡在送出
		if r.err != nil {
			if firstErr == nil {
				firstErr = r.err
				cancel()
			}
			continue
		}
		titles[r.index] = r.title
	}
	if firstErr != nil {
		return nil, firstErr
	}
	return titles, nil
}

// ANCHOR_END: pool
