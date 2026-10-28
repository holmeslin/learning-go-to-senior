package orders_test

import (
	"context"
	"database/sql"
	"fmt"
	"sync/atomic"
	"time"

	_ "modernc.org/sqlite"

	orders "github.com/holmeslin/learning-go-to-senior/examples/chapter4/04_pool_size/before"
)

var dbSeq atomic.Int64

// newDB 開一個只屬於這次呼叫的記憶體資料庫，放進三筆訂單。
// 名稱唯一的共享快取 DSN 讓池子裡的每條連線看到同一份資料。maxOpen 為 0 表示不設上限。
func newDB(maxOpen int) *sql.DB {
	dsn := fmt.Sprintf("file:orders_before_%d?mode=memory&cache=shared", dbSeq.Add(1))
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		panic(err)
	}
	db.SetMaxOpenConns(maxOpen)
	_, err = db.Exec(`CREATE TABLE orders (id INTEGER PRIMARY KEY, customer TEXT NOT NULL, total INTEGER NOT NULL);
		INSERT INTO orders (customer, total) VALUES ('ann', 120), ('bob', 980), ('cat', 450)`)
	if err != nil {
		panic(err)
	}
	return db
}

// ANCHOR: small

func ExampleStore_Find_smallPool() {
	db := newDB(1) // 池子只有一條連線
	defer db.Close()
	store := orders.NewStore(db)

	// 風控規則：金額超過 500 的第一筆訂單。Find 在迴圈中途回傳。
	o, err := store.Find(context.Background(), func(o orders.Order) bool { return o.Total > 500 })
	fmt.Println(o.ID, err)

	// 下一個查詢帶著 200 毫秒的 deadline，等不到連線。
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	var n int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM orders").Scan(&n)
	fmt.Println(err)

	st := db.Stats()
	fmt.Printf("in use: %d, wait count: %d\n", st.InUse, st.WaitCount)
	// Output:
	// 2 <nil>
	// context deadline exceeded
	// in use: 1, wait count: 1
}

// ANCHOR_END: small

// ANCHOR: unlimited

func ExampleStore_Find_unlimitedPool() {
	db := newDB(0) // 不設上限：預設值
	defer db.Close()
	store := orders.NewStore(db)

	for range 3 {
		_, err := store.Find(context.Background(), func(o orders.Order) bool { return o.Total > 500 })
		if err != nil {
			fmt.Println(err)
		}
	}

	st := db.Stats()
	fmt.Printf("open: %d, in use: %d, wait count: %d\n", st.OpenConnections, st.InUse, st.WaitCount)
	// Output: open: 3, in use: 3, wait count: 0
}

// ANCHOR_END: unlimited
