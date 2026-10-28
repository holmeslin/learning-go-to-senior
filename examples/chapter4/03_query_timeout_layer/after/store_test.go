package orders_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	orders "github.com/holmeslin/learning-go-to-senior/examples/chapter4/03_query_timeout_layer/after"
)

func TestStoreCounts(t *testing.T) {
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
func TestPendingCountCallerContext(t *testing.T) {
	tests := []struct {
		name string
		ctx  func(*testing.T) context.Context
		want error
	}{
		{
			name: "canceled",
			ctx: func(t *testing.T) context.Context {
				ctx, cancel := context.WithCancel(t.Context())
				cancel()
				return ctx
			},
			want: context.Canceled,
		},
		{
			name: "deadline",
			ctx: func(t *testing.T) context.Context {
				ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
				t.Cleanup(cancel)
				return ctx
			},
			want: context.DeadlineExceeded,
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
		})
	}
}

// ANCHOR_END: test

// store 的上限是從呼叫端的 context 縮短來的：呼叫端沒有 deadline 時，store 的上限先到。
func TestSearchCountOwnLimit(t *testing.T) {
	db := newDB(t)
	busy(t, db)
	store := orders.NewStore(db, 100*time.Millisecond)

	_, err := store.SearchCount(t.Context(), "gift")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("err = %v; want it to wrap context.DeadlineExceeded", err)
	}
}

// 縮短只會更早，不會更晚：呼叫端的 deadline 比 store 的上限早，就由呼叫端決定。
func TestSearchCountNeverExtends(t *testing.T) {
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
func TestSearchCountCanceled(t *testing.T) {
	store := orders.NewStore(newDB(t), time.Hour)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := store.SearchCount(ctx, "gift")
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v; want it to wrap context.Canceled", err)
	}
}

// 取消 BeginTx 收到的 ctx，交易就被回滾：寫入沒有留下來，Commit 也回傳錯誤。
func TestBeginTxCancelRollsBack(t *testing.T) {
	db := newDB(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx() error = %v", err)
	}
	if _, err := tx.ExecContext(t.Context(), `INSERT INTO orders (status, note) VALUES ('pending', 'in tx')`); err != nil {
		t.Fatalf("insert in tx: %v", err)
	}

	cancel()

	// 唯一的連線要等回滾做完才會放開，所以這個查詢看到的一定是回滾之後的資料。
	// 5 秒只是保險：交易沒被回滾的話，查詢拿不到連線，測試以逾時失敗，而不是卡住。
	guard, stop := context.WithTimeout(t.Context(), 5*time.Second)
	defer stop()
	var n int
	if err := db.QueryRowContext(guard, `SELECT COUNT(*) FROM orders WHERE status = 'pending'`).Scan(&n); err != nil || n != 3 {
		t.Errorf("pending orders = %d, %v; want 3, <nil> (the insert rolled back)", n, err)
	}
	if err := tx.Commit(); !errors.Is(err, sql.ErrTxDone) {
		t.Errorf("Commit() error = %v; want %v", err, sql.ErrTxDone)
	}
}
