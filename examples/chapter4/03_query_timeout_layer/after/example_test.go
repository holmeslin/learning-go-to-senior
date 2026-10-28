package orders_test

import (
	"context"
	"fmt"
	"time"

	orders "github.com/holmeslin/learning-go-to-senior/examples/chapter4/03_query_timeout_layer/after"
)

// ANCHOR: example

func ExampleStore_PendingCount() {
	db := openDB()
	defer db.Close()
	store := orders.NewStore(db, time.Minute)

	tx := hold(db) // 另一個交易占住唯一的連線：資料庫「忙碌」，查詢只能排隊
	defer tx.Rollback()

	// 呼叫端取消了：查詢不再排隊。
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := store.PendingCount(canceled)
	fmt.Println(err)

	// 呼叫端的 deadline 到了：已經在排隊的查詢跟著放棄。
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err = store.PendingCount(ctx)
	fmt.Println(err)
	// Output:
	// count pending orders: context canceled
	// count pending orders: context deadline exceeded
}

// ANCHOR_END: example
