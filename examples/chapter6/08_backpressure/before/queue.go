// Package thumbnail 是縮圖服務的工作佇列（before：producer 與 consumer 之間沒有上限）。
package thumbnail

import "sync"

// Job 是一張要縮圖的圖片。
type Job struct {
	// Path 是圖片的路徑。
	Path string
}

// ANCHOR: queue

// Queue 把縮圖工作交給一個背景的 consumer 依序處理。
type Queue struct {
	mu   sync.Mutex
	jobs []Job
	wake chan struct{} // 容量 1：通知 consumer 佇列裡有東西
	done chan struct{}
}

// New 建立佇列並啟動 consumer；consumer 對每一筆工作呼叫 process。
func New(process func(Job)) *Queue {
	q := &Queue{wake: make(chan struct{}, 1), done: make(chan struct{})}
	go q.run(process)
	return q
}

// Submit 把 j 排進佇列，立刻回傳；佇列沒有長度上限。
func (q *Queue) Submit(j Job) {
	q.mu.Lock()
	q.jobs = append(q.jobs, j)
	q.mu.Unlock()
	select {
	case q.wake <- struct{}{}:
	default: // 已經有通知在等了
	}
}

// ANCHOR_END: queue

// Pending 回傳排隊中、還沒被 consumer 取走的工作數。
func (q *Queue) Pending() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.jobs)
}

// Close 讓 consumer 把佇列裡的工作做完後結束，並等它結束。
// 呼叫之後不能再 Submit。
func (q *Queue) Close() {
	close(q.wake)
	<-q.done
}

func (q *Queue) run(process func(Job)) {
	defer close(q.done)
	for range q.wake {
		for {
			j, ok := q.pop()
			if !ok {
				break
			}
			process(j)
		}
	}
}

func (q *Queue) pop() (Job, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.jobs) == 0 {
		return Job{}, false
	}
	j := q.jobs[0]
	q.jobs = q.jobs[1:]
	return j, true
}
