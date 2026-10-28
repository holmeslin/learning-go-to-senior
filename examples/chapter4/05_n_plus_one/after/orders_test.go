package orders_test

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"testing"

	orders "github.com/holmeslin/learning-go-to-senior/examples/chapter4/05_n_plus_one/after"
)

func newDB(t *testing.T, fixtures ...string) *sql.DB {
	t.Helper()
	db, err := openDB(fixtures...)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close db: %v", err)
		}
	})
	return db
}

func TestRecentOrdersAssemblesItems(t *testing.T) {
	store := orders.NewStore(newDB(t, seed))

	got, err := store.RecentOrders(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}

	// 最新的在前；第 3 張沒有品項，仍然要在結果裡，Items 是空的；品項依新增順序。
	want := []orders.Order{
		{ID: 4, Customer: "dan", Items: []orders.Item{{SKU: "cap", Qty: 1}, {SKU: "tape", Qty: 3}, {SKU: "glue", Qty: 2}}},
		{ID: 3, Customer: "cara"},
		{ID: 2, Customer: "bob", Items: []orders.Item{{SKU: "pad", Qty: 5}}},
		{ID: 1, Customer: "ann", Items: []orders.Item{{SKU: "pen", Qty: 2}, {SKU: "ink", Qty: 1}}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("RecentOrders = %+v\nwant %+v", got, want)
	}
}

// limit 只取最近的幾張，其他訂單的品項不能混進來。
func TestRecentOrdersRespectsLimit(t *testing.T) {
	store := orders.NewStore(newDB(t, seed))

	got, err := store.RecentOrders(context.Background(), 2)
	if err != nil {
		t.Fatal(err)
	}

	want := []orders.Order{
		{ID: 4, Customer: "dan", Items: []orders.Item{{SKU: "cap", Qty: 1}, {SKU: "tape", Qty: 3}, {SKU: "glue", Qty: 2}}},
		{ID: 3, Customer: "cara"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("RecentOrders = %+v\nwant %+v", got, want)
	}
}

// 查詢次數不隨訂單數成長；沒有訂單時不送第二次查詢（空的 IN () 不是標準 SQL）。
func TestRecentOrdersQueryCount(t *testing.T) {
	for _, tc := range []struct {
		orders      int
		wantQueries int
	}{
		{orders: 0, wantQueries: 1},
		{orders: 1, wantQueries: 2},
		{orders: 25, wantQueries: 2},
	} {
		t.Run(fmt.Sprintf("%d orders", tc.orders), func(t *testing.T) {
			db := newDB(t)
			for i := 1; i <= tc.orders; i++ {
				if _, err := db.Exec(`INSERT INTO orders (id, customer) VALUES (?, 'ann')`, i); err != nil {
					t.Fatal(err)
				}
				if _, err := db.Exec(`INSERT INTO order_items (order_id, sku, qty) VALUES (?, 'pen', 1), (?, 'ink', 2)`, i, i); err != nil {
					t.Fatal(err)
				}
			}

			q := &countingQuerier{Querier: db}
			got, err := orders.NewStore(q).RecentOrders(context.Background(), 1000)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != tc.orders {
				t.Errorf("got %d orders; want %d", len(got), tc.orders)
			}
			if q.n != tc.wantQueries {
				t.Errorf("queries = %d; want %d", q.n, tc.wantQueries)
			}
		})
	}
}
