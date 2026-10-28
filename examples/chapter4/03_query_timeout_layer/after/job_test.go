package orders_test

import (
	"context"
	"errors"
	"testing"
	"time"

	orders "github.com/holmeslin/learning-go-to-senior/examples/chapter4/03_query_timeout_layer/after"
)

func TestRunPendingJob(t *testing.T) {
	store := orders.NewStore(newDB(t), time.Hour)

	if n, err := orders.RunPendingJob(t.Context(), store, time.Minute); err != nil || n != 3 {
		t.Errorf("RunPendingJob = %d, %v; want 3, <nil>", n, err)
	}
}

// 背景工作沒有請求可以繼承，deadline 是它自己設的。
func TestRunPendingJob_ownDeadline(t *testing.T) {
	db := newDB(t)
	busy(t, db)
	store := orders.NewStore(db, time.Hour)

	_, err := orders.RunPendingJob(t.Context(), store, 10*time.Millisecond)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("err = %v; want it to wrap context.DeadlineExceeded", err)
	}
}

// 整個程式關機時（上層的 context 取消），進行中的工作跟著停。
func TestRunPendingJob_shutdown(t *testing.T) {
	store := orders.NewStore(newDB(t), time.Hour)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := orders.RunPendingJob(ctx, store, time.Minute)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v; want it to wrap context.Canceled", err)
	}
}
