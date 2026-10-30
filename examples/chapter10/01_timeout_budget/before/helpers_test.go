package budget_test

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	budget "github.com/holmeslin/learning-go-to-senior/examples/chapter10/01_timeout_budget/before"
)

// ANCHOR: helpers

// recorder 記下每個下游呼叫的開始、結束與當時的 deadline，
// 時間是 synctest bubble 裡的假時間，所以每一筆都是確定的。
type recorder struct {
	mu    sync.Mutex
	start time.Time
	log   []string
}

func newRecorder() *recorder { return &recorder{start: time.Now()} }

func (r *recorder) add(format string, args ...any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.log = append(r.log, fmt.Sprintf("%v %s", time.Since(r.start), fmt.Sprintf(format, args...)))
}

func (r *recorder) entries() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.log)
}

// backend 模擬一個要花 d 的下游：做完就記 done，ctx 先結束就記 aborted。
func (r *recorder) backend(name string, d time.Duration) budget.Backend {
	return func(ctx context.Context) error {
		left := "none"
		if deadline, ok := ctx.Deadline(); ok {
			left = time.Until(deadline).String()
		}
		r.add("%s start, deadline in %s", name, left)

		t := time.NewTimer(d)
		defer t.Stop()
		select {
		case <-t.C:
			r.add("%s done", name)
			return nil
		case <-ctx.Done():
			r.add("%s aborted", name)
			return ctx.Err()
		}
	}
}

// ANCHOR_END: helpers
