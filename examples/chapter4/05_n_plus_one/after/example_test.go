package orders_test

import (
	"context"
	"database/sql"
	"fmt"

	orders "github.com/holmeslin/learning-go-to-senior/examples/chapter4/05_n_plus_one/after"
)

// countingQuerier 包住 [orders.Querier]，數出 QueryContext 被呼叫了幾次。
type countingQuerier struct {
	orders.Querier
	n int
}

func (c *countingQuerier) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	c.n++
	return c.Querier.QueryContext(ctx, query, args...)
}

// ANCHOR: example

func ExampleStore_RecentOrders() {
	db, err := openDB(seed)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer db.Close()

	var got []orders.Order
	for _, limit := range []int{1, 2, 4} {
		q := &countingQuerier{Querier: db}
		got, err = orders.NewStore(q).RecentOrders(context.Background(), limit)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Printf("%d 張訂單，%d 次查詢\n", len(got), q.n)
	}
	// Output:
	// 1 張訂單，2 次查詢
	// 2 張訂單，2 次查詢
	// 4 張訂單，2 次查詢
}

// ANCHOR_END: example
