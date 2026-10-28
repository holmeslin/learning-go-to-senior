// Package thumbnail 是縮圖服務的工作佇列（after：有上限，滿了就等或拒絕）。
package thumbnail

import (
	"context"
	"errors"
)

// Job 是一張要縮圖的圖片。
type Job struct {
	// Path 是圖片的路徑。
	Path string
}

// ErrFull 表示佇列已滿，工作沒有被接受。
var ErrFull = errors.New("queue full")

// ANCHOR: queue

// Queue 把縮圖工作交給一個背景的 consumer 依序處理，排隊的數量有上限。
type Queue struct {
	jobs chan Job
	done chan struct{}
}

// New 建立最多排隊 capacity 筆的佇列並啟動 consumer；consumer 對每一筆工作呼叫 process。
// capacity 為負數時 panic，與 make 相同。
func New(capacity int, process func(Job)) *Queue {
	q := &Queue{jobs: make(chan Job, capacity), done: make(chan struct{})}
	go func() {
		defer close(q.done)
		for j := range q.jobs {
			process(j)
		}
	}()
	return q
}

// Submit 把 j 排進佇列。佇列滿了就等到有空位或 ctx 結束，
// 後者回傳 ctx.Err()：壓力會順著呼叫端往上傳。
func (q *Queue) Submit(ctx context.Context, j Job) error {
	select {
	case q.jobs <- j:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// TrySubmit 把 j 排進佇列，不等待。佇列滿了回傳 ErrFull，工作不會被排進去。
func (q *Queue) TrySubmit(j Job) error {
	select {
	case q.jobs <- j:
		return nil
	default:
		return ErrFull
	}
}

// ANCHOR_END: queue

// Pending 回傳排隊中、還沒被 consumer 取走的工作數，不會超過 capacity。
func (q *Queue) Pending() int { return len(q.jobs) }

// Close 讓 consumer 把佇列裡的工作做完後結束，並等它結束。
// 呼叫之後不能再 Submit 或 TrySubmit，也不能與它們同時呼叫。
func (q *Queue) Close() {
	close(q.jobs)
	<-q.done
}
