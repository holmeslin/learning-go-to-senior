package bank_test

import (
	"context"
	"database/sql"
	"fmt"
	"sync/atomic"

	_ "modernc.org/sqlite"

	bank "github.com/holmeslin/learning-go-to-senior/examples/chapter4/02_transaction_boundary/before"
)

// dbSeq 讓每個 Example 開出名字不同的資料庫。
var dbSeq atomic.Int64

// newDB 開一個全新的 in-memory 資料庫，建好資料表，放入 ann=500、bob=100。
// DSN 是具名的共享快取：連線池開出第二條連線時，看到的仍是同一個資料庫。
func newDB() *sql.DB {
	dsn := fmt.Sprintf("file:before%d?mode=memory&cache=shared", dbSeq.Add(1))
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

// ANCHOR: example

func ExampleService_Transfer() {
	db := newDB()
	defer db.Close()
	svc := bank.NewService(bank.NewStore(db))

	// nobody 這個帳戶不存在：扣款先提交了，入帳才失敗。
	err := svc.Transfer(context.Background(), "ann", "nobody", 300)
	balances, ledger := readState(db)

	fmt.Println("transfer:", err)
	fmt.Println("balances:", balances)
	fmt.Println("ledger:", ledger)
	// Output:
	// transfer: credit nobody: account not found
	// balances: map[ann:200 bob:100]
	// ledger: [ann:-300]
}

// ANCHOR_END: example
