package orders_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	orders "github.com/holmeslin/learning-go-to-senior/examples/chapter4/04_pool_size/after"
)

// guard 讓「連線沒歸還」的 bug 以逾時失敗收場，而不是讓測試卡住。
// 正常情況下沒有任何呼叫會等到它，所以它的長短不影響結果。
const guard = 5 * time.Second

// Find 的每一條離開路徑（中途找到、全部讀完、Scan 失敗）都要歸還連線。
// 池子只有一條連線，連續呼叫三次：只要有一次沒歸還，下一次就拿不到連線。
func TestFindReturnsConnection(t *testing.T) {
	never := func(orders.Order) bool { return false }
	tests := []struct {
		name    string
		seed    string // 額外執行的 SQL
		match   func(orders.Order) bool
		wantErr bool
	}{
		{"match in the middle", "", func(o orders.Order) bool { return o.Total > 500 }, false},
		{"no match", "", never, true},
		{"scan error", "INSERT INTO orders (customer, total) VALUES ('dan', 'n/a')", never, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db := newDB(1)
			defer db.Close()
			if tt.seed != "" {
				if _, err := db.Exec(tt.seed); err != nil {
					t.Fatalf("seed: %v", err)
				}
			}
			store := orders.NewStore(db)
			ctx, cancel := context.WithTimeout(context.Background(), guard)
			defer cancel()

			for i := range 3 {
				_, err := store.Find(ctx, tt.match)
				if (err != nil) != tt.wantErr {
					t.Fatalf("Find() #%d error = %v, wantErr %v", i+1, err, tt.wantErr)
				}
			}

			if st := db.Stats(); st.InUse != 0 || st.WaitCount != 0 {
				t.Errorf("InUse = %d, WaitCount = %d, want both 0", st.InUse, st.WaitCount)
			}
		})
	}
}

func TestFindReportsNotFound(t *testing.T) {
	t.Parallel()
	db := newDB(1)
	defer db.Close()

	_, err := orders.NewStore(db).Find(context.Background(), func(orders.Order) bool { return false })

	if !errors.Is(err, orders.ErrNotFound) {
		t.Errorf("Find() error = %v, want ErrNotFound", err)
	}
}

// maxOpen 小於 1 在 database/sql 裡代表「不設上限」，正是要避免的設定，所以 Open 要拒絕。
func TestOpenRejectsUnlimitedPool(t *testing.T) {
	t.Parallel()
	for _, n := range []int{0, -1} {
		db, err := orders.Open("file:unused?mode=memory&cache=shared", n)
		if err == nil {
			db.Close()
			t.Errorf("Open(maxOpen=%d) error = nil, want an error", n)
		}
	}
}

func TestOpenSetsPool(t *testing.T) {
	t.Parallel()
	db := newDB(3)
	defer db.Close()
	if got := db.Stats().MaxOpenConnections; got != 3 {
		t.Errorf("MaxOpenConnections = %d, want 3", got)
	}

	// 同時占住三條連線再全部歸還：閒置上限和池子一樣大，三條都該留著。
	// database/sql 預設只留兩條，多出來的那條會被關掉，MaxIdleClosed 就是 1。
	var held []*sql.Rows
	for range 3 {
		rows, err := db.Query("SELECT id FROM orders")
		if err != nil {
			t.Fatalf("Query() error = %v", err)
		}
		t.Cleanup(func() { _ = rows.Close() })
		held = append(held, rows)
	}
	if st := db.Stats(); st.InUse != 3 {
		t.Fatalf("InUse = %d, want 3", st.InUse)
	}
	for _, rows := range held {
		if err := rows.Close(); err != nil {
			t.Errorf("rows.Close() error = %v", err)
		}
	}

	if st := db.Stats(); st.Idle != 3 || st.MaxIdleClosed != 0 {
		t.Errorf("Idle = %d, MaxIdleClosed = %d, want 3 and 0", st.Idle, st.MaxIdleClosed)
	}
}
