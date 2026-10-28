package orders

import (
	"context"
	"time"
)

// ANCHOR: job

// RunPendingJob 是背景工作的一次執行：統計待處理的訂單數。
// 背景工作沒有請求可以繼承 deadline，所以自己設 limit；
// ctx 是整個程式的 context，關機時取消，進行中的查詢也會跟著停。
func RunPendingJob(ctx context.Context, store *Store, limit time.Duration) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, limit)
	defer cancel()

	return store.PendingCount(ctx)
}

// ANCHOR_END: job
