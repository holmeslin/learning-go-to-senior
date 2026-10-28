package fetch

import (
	"context"
	"errors"
	"sync"
)

// ANCHOR: join

// FetchAllJoin 同時抓取 srcs 裡的每個來源，等全部結束後，用 errors.Join 回傳所有錯誤。
// 某個來源失敗不會取消其他來源。
func FetchAllJoin(ctx context.Context, srcs []string, fetch Fetch) error {
	errs := make([]error, len(srcs))
	var wg sync.WaitGroup
	for i, src := range srcs {
		wg.Go(func() { errs[i] = fetch(ctx, src) })
	}
	wg.Wait()
	return errors.Join(errs...)
}

// ANCHOR_END: join
