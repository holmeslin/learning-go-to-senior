// Package orders 讀取訂單與它的品項。
package orders

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"
)

// Order 是一張訂單。沒有任何品項的訂單，Items 是 nil。
type Order struct {
	ID       int
	Customer string
	Items    []Item
}

// Item 是訂單裡的一個品項。
type Item struct {
	SKU string
	Qty int
}

// Querier 是 [Store] 需要的查詢能力，*sql.DB 與 *sql.Tx 都符合。
type Querier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// Store 從資料庫讀取訂單。
type Store struct {
	q Querier
}

// NewStore 回傳用 q 查詢的 Store。
func NewStore(q Querier) *Store { return &Store{q: q} }

// ANCHOR: recent

// RecentOrders 回傳最近的 limit 張訂單（編號大的在前），每張帶著自己的品項。
// 不論有幾張訂單，最多只查兩次。
func (s *Store) RecentOrders(ctx context.Context, limit int) ([]Order, error) {
	orders, err := s.listOrders(ctx, limit)
	if err != nil {
		return nil, err
	}
	if len(orders) == 0 {
		return orders, nil // 沒有訂單就不用查品項；空的 IN () 不是標準 SQL，SQLite 放行，別的資料庫未必
	}

	ids := make([]int, len(orders))
	for i, o := range orders {
		ids[i] = o.ID
	}
	byOrder, err := s.itemsByOrder(ctx, ids)
	if err != nil {
		return nil, err
	}
	for i := range orders {
		orders[i].Items = byOrder[orders[i].ID]
	}
	return orders, nil
}

// ANCHOR_END: recent

func (s *Store) listOrders(ctx context.Context, limit int) ([]Order, error) {
	rows, err := s.q.QueryContext(ctx, `SELECT id, customer FROM orders ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("list orders: %w", err)
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var o Order
		if err := rows.Scan(&o.ID, &o.Customer); err != nil {
			return nil, fmt.Errorf("scan order: %w", err)
		}
		orders = append(orders, o)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list orders: %w", err)
	}
	return orders, nil
}

// ANCHOR: itemsbyorder

// itemsByOrder 一次查出 ids 這幾張訂單的所有品項，依訂單編號分組。
func (s *Store) itemsByOrder(ctx context.Context, ids []int) (map[int][]Item, error) {
	// 每個值一個 ?，值本身走參數；SQL 字串裡只有 ?，沒有任何資料。
	marks := strings.Join(slices.Repeat([]string{"?"}, len(ids)), ", ")
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}

	rows, err := s.q.QueryContext(ctx,
		`SELECT order_id, sku, qty FROM order_items WHERE order_id IN (`+marks+`) ORDER BY id`, args...)
	if err != nil {
		return nil, fmt.Errorf("items of %d orders: %w", len(ids), err)
	}
	defer rows.Close()

	byOrder := make(map[int][]Item, len(ids))
	for rows.Next() {
		var orderID int
		var it Item
		if err := rows.Scan(&orderID, &it.SKU, &it.Qty); err != nil {
			return nil, fmt.Errorf("scan item: %w", err)
		}
		byOrder[orderID] = append(byOrder[orderID], it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("items of %d orders: %w", len(ids), err)
	}
	return byOrder, nil
}

// ANCHOR_END: itemsbyorder
