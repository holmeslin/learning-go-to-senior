package orders_test

import (
	"context"
	"errors"
	"testing"
	"time"

	orders "github.com/holmeslin/learning-go-to-senior/examples/chapter4/03_query_timeout_layer/after"
)

func TestStore_counts(t *testing.T) {
	store := orders.NewStore(newDB(t), time.Hour)

	if n, err := store.PendingCount(t.Context()); err != nil || n != 3 {
		t.Errorf("PendingCount = %d, %v; want 3, <nil>", n, err)
	}
	if n, err := store.SearchCount(t.Context(), "gift"); err != nil || n != 2 {
		t.Errorf("SearchCount = %d, %v; want 2, <nil>", n, err)
	}
}

// ANCHOR: test

// 資料庫忙碌（唯一的連線被占住）時，查詢只能排隊。
// 呼叫端的取消與 deadline 都要讓它回傳對應的錯誤。
func TestStore_PendingCount_callerContext(t *testing.T) {
	tests := []struct {
		name      string
		ctx       func(*testing.T) context.Context
		want      error
		wantWaits int64 // 查詢有沒有排進等連線的隊伍
	}{
		{
			name: "canceled",
			ctx: func(t *testing.T) context.Context {
				ctx, cancel := context.WithCancel(t.Context())
				cancel()
				return ctx
			},
			want:      context.Canceled,
			wantWaits: 0,
		},
		{
			name: "deadline",
			ctx: func(t *testing.T) context.Context {
				ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
				t.Cleanup(cancel)
				return ctx
			},
			want:      context.DeadlineExceeded,
			wantWaits: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newDB(t)
			busy(t, db)
			store := orders.NewStore(db, time.Hour)

			_, err := store.PendingCount(tt.ctx(t))
			if !errors.Is(err, tt.want) {
				t.Errorf("err = %v; want it to wrap %v", err, tt.want)
			}
			if got := db.Stats().WaitCount; got != tt.wantWaits {
				t.Errorf("WaitCount = %d; want %d", got, tt.wantWaits)
			}
		})
	}
}

// ANCHOR_END: test

// store 的上限是從呼叫端的 context 縮短來的：呼叫端沒有 deadline 時，store 的上限先到。
func TestStore_SearchCount_ownLimit(t *testing.T) {
	db := newDB(t)
	busy(t, db)
	store := orders.NewStore(db, 100*time.Millisecond)

	_, err := store.SearchCount(t.Context(), "gift")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("err = %v; want it to wrap context.DeadlineExceeded", err)
	}
}

// 縮短只會更早，不會更晚：呼叫端的 deadline 比 store 的上限早，就由呼叫端決定。
func TestStore_SearchCount_neverExtends(t *testing.T) {
	db := newDB(t)
	busy(t, db)
	store := orders.NewStore(db, time.Hour)

	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	_, err := store.SearchCount(ctx, "gift")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("err = %v; want it to wrap context.DeadlineExceeded", err)
	}
}

// 縮短之後，呼叫端的取消照樣傳得進去。
func TestStore_SearchCount_canceled(t *testing.T) {
	store := orders.NewStore(newDB(t), time.Hour)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := store.SearchCount(ctx, "gift")
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v; want it to wrap context.Canceled", err)
	}
}
