package budget_test

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	budget "github.com/holmeslin/learning-go-to-senior/examples/chapter10/01_timeout_budget/after"
)

// recorder 與 before 的相同：記下每個下游呼叫的開始、結束與當時的 deadline。
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
