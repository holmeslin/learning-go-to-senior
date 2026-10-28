package orders_test

import (
	"context"
	"database/sql"
	"fmt"
	"sync/atomic"
	"time"

	orders "github.com/holmeslin/learning-go-to-senior/examples/chapter4/04_pool_size/after"
)

var dbSeq atomic.Int64

// newDB 開一個只屬於這次呼叫的記憶體資料庫，放進三筆訂單，池子由 orders.Open 設定。
// 名稱唯一的共享快取 DSN 讓池子裡的每條連線看到同一份資料。
func newDB(maxOpen int) *sql.DB {
	dsn := fmt.Sprintf("file:orders_after_%d?mode=memory&cache=shared", dbSeq.Add(1))
	db, err := orders.Open(dsn, maxOpen)
	if err != nil {
		panic(err)
	}
	_, err = db.Exec(`CREATE TABLE orders (id INTEGER PRIMARY KEY, customer TEXT NOT NULL, total INTEGER NOT NULL);
		INSERT INTO orders (customer, total) VALUES ('ann', 120), ('bob', 980), ('cat', 450)`)
	if err != nil {
		panic(err)
	}
	return db
}

// ANCHOR: example

func ExampleStore_Find() {
	db := newDB(1) // 池子同樣只有一條連線
	defer db.Close()
	store := orders.NewStore(db)

	o, err := store.Find(context.Background(), func(o orders.Order) bool { return o.Total > 500 })
	fmt.Println(o.ID, err)

	// 連線已經歸還：同樣的查詢馬上拿到連線，不必等。deadline 給得很寬，只是為了萬一沒歸還時不要卡住。
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var n int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM orders").Scan(&n)
	fmt.Println(n, err)

	st := db.Stats()
	fmt.Printf("in use: %d, wait count: %d\n", st.InUse, st.WaitCount)
	// Output:
	// 2 <nil>
	// 3 <nil>
	// in use: 0, wait count: 0
}

// ANCHOR_END: example
