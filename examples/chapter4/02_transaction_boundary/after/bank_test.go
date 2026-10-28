package bank_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"maps"
	"slices"
	"sync/atomic"
	"testing"

	_ "modernc.org/sqlite"

	bank "github.com/holmeslin/learning-go-to-senior/examples/chapter4/02_transaction_boundary/after"
)

// 編譯期檢查：*sql.DB 與 *sql.Tx 都符合 bank.Execer。
var (
	_ bank.Execer = (*sql.DB)(nil)
	_ bank.Execer = (*sql.Tx)(nil)
)

// dbSeq 讓每個測試與 Example 開出名字不同的資料庫。
var dbSeq atomic.Int64

// newDB 開一個全新的 in-memory 資料庫，建好資料表，放入 ann=500、bob=100。
// DSN 是具名的共享快取：連線池開出第二條連線時，看到的仍是同一個資料庫。
func newDB() *sql.DB {
	dsn := fmt.Sprintf("file:after%d?mode=memory&cache=shared", dbSeq.Add(1))
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		panic(err)
	}
	seed := bank.Schema + `INSERT INTO accounts VALUES ('ann', 500), ('bob', 100);`
	if _, err := db.ExecContext(context.Background(), seed); err != nil {
		panic(err)
	}
	return db
}

// newTestDB 是測試用的 newDB，結束時關閉資料庫並檢查錯誤。
func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db := newDB()
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close db: %v", err)
		}
	})
	return db
}

// readState 讀出每個帳戶的餘額，以及帳本（每次餘額異動各記一筆）。
func readState(db *sql.DB) (balances map[string]int, ledger []string) {
	ctx := context.Background()
	balances = map[string]int{}

	rows, err := db.QueryContext(ctx, `SELECT name, balance FROM accounts`)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		var balance int
		if err := rows.Scan(&name, &balance); err != nil {
			panic(err)
		}
		balances[name] = balance
	}
	if err := rows.Err(); err != nil {
		panic(err)
	}

	entries, err := db.QueryContext(ctx, `SELECT account, delta FROM entries ORDER BY rowid`)
	if err != nil {
		panic(err)
	}
	defer entries.Close()
	for entries.Next() {
		var account string
		var delta int
		if err := entries.Scan(&account, &delta); err != nil {
			panic(err)
		}
		ledger = append(ledger, fmt.Sprintf("%s:%+d", account, delta))
	}
	if err := entries.Err(); err != nil {
		panic(err)
	}
	return balances, ledger
}

// wantState 比對資料庫現在的狀態。
func wantState(t *testing.T, db *sql.DB, wantBalances map[string]int, wantLedger []string) {
	t.Helper()
	balances, ledger := readState(db)
	if !maps.Equal(balances, wantBalances) {
		t.Errorf("balances = %v, want %v", balances, wantBalances)
	}
	if !slices.Equal(ledger, wantLedger) {
		t.Errorf("ledger = %v, want %v", ledger, wantLedger)
	}
}

func TestTransferWritesBothSides(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	svc := bank.NewService(db)

	if err := svc.Transfer(context.Background(), "ann", "bob", 300); err != nil {
		t.Fatalf("Transfer() error = %v", err)
	}

	wantState(t, db, map[string]int{"ann": 200, "bob": 400}, []string{"ann:-300", "bob:+300"})
}

func TestTransferFailureLeavesNothingBehind(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		to      string
		amount  int
		wantErr error // nil 表示只要求有錯誤，不要求是哪一個
	}{
		{name: "credit step fails", to: "nobody", amount: 300, wantErr: bank.ErrNotFound},
		{name: "debit step fails", to: "bob", amount: 600},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db := newTestDB(t)
			svc := bank.NewService(db)

			err := svc.Transfer(context.Background(), "ann", tt.to, tt.amount)

			if err == nil {
				t.Fatal("Transfer() error = nil, want an error")
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Errorf("Transfer() error = %v, want it to wrap %v", err, tt.wantErr)
			}
			wantState(t, db, map[string]int{"ann": 500, "bob": 100}, nil)
		})
	}
}

// Store 綁在 *sql.DB 上時也能直接用：同一組方法在交易內外重用。
func TestStoreWorksWithoutTransaction(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)

	if err := bank.NewStore(db).Credit(context.Background(), "bob", 50); err != nil {
		t.Fatalf("Credit() error = %v", err)
	}

	wantState(t, db, map[string]int{"ann": 500, "bob": 150}, []string{"bob:+50"})
}
