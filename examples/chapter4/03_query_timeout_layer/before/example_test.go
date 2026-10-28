package orders_test

import (
	"context"
	"fmt"
	"time"

	orders "github.com/holmeslin/learning-go-to-senior/examples/chapter4/03_query_timeout_layer/before"
)

// ANCHOR: example

func ExampleStore_PendingCount() {
	db := openDB()
	defer db.Close()
	store := orders.NewStore(db, 50*time.Millisecond)

	// 呼叫端取消了。為了讓輸出固定，這裡先取消再呼叫。
	canceled, cancel := context.WithCancel(context.Background())
	cancel()

	// 資料庫正常：已經沒人在等的查詢照跑，還回傳結果。
	n, err := store.PendingCount(canceled)
	fmt.Println(canceled.Err(), n, err)

	// 資料庫忙碌（另一個交易占住唯一的連線）：查詢照樣排隊，等到 store 自己的上限才放棄。
	tx := hold(db)
	defer tx.Rollback()
	_, err = store.PendingCount(canceled)
	fmt.Println(err, db.Stats().WaitCount)
	// Output:
	// context canceled 3 <nil>
	// count pending orders: context deadline exceeded 1
}

// ANCHOR_END: example
