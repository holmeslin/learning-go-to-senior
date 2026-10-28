// Package orders 讀取訂單與它的品項。
package orders

import (
	"context"
	"database/sql"
	"fmt"
)

// Order 是一張訂單。
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

// ANCHOR: querier

// Querier 是 [Store] 需要的查詢能力，*sql.DB 與 *sql.Tx 都符合。
type Querier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// ANCHOR_END: querier

// Store 從資料庫讀取訂單。
type Store struct {
	q Querier
}

// NewStore 回傳用 q 查詢的 Store。
func NewStore(q Querier) *Store { return &Store{q: q} }

// ANCHOR: recent

// RecentOrders 回傳最近的 limit 張訂單（編號大的在前），每張帶著自己的品項。
func (s *Store) RecentOrders(ctx context.Context, limit int) ([]Order, error) {
	orders, err := s.listOrders(ctx, limit)
	if err != nil {
		return nil, err
	}
	for i := range orders {
		items, err := s.itemsOf(ctx, orders[i].ID)
		if err != nil {
			return nil, err
		}
		orders[i].Items = items
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

// ANCHOR: itemsof

// itemsOf 查一張訂單的品項。
func (s *Store) itemsOf(ctx context.Context, orderID int) ([]Item, error) {
	rows, err := s.q.QueryContext(ctx, `SELECT sku, qty FROM order_items WHERE order_id = ? ORDER BY id`, orderID)
	if err != nil {
		return nil, fmt.Errorf("items of order %d: %w", orderID, err)
	}
	defer rows.Close()

	var items []Item
	for rows.Next() {
		var it Item
		if err := rows.Scan(&it.SKU, &it.Qty); err != nil {
			return nil, fmt.Errorf("scan item of order %d: %w", orderID, err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("items of order %d: %w", orderID, err)
	}
	return items, nil
}

// ANCHOR_END: itemsof
