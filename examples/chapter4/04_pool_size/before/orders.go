// Package orders 查詢訂單；Find 在迴圈中途回傳時沒有關閉 rows，會占住連線。
package orders

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// ErrNotFound 表示沒有符合條件的訂單。
var ErrNotFound = errors.New("order not found")

// Order 是一筆訂單。
type Order struct {
	ID       int
	Customer string
	Total    int
}

// Store 從資料庫讀取訂單。
type Store struct {
	db *sql.DB
}

// NewStore 回傳使用 db 的 Store。
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// ANCHOR: find

// Find 依 id 順序找出第一筆讓 match 回傳 true 的訂單；沒有符合的回傳 ErrNotFound。
func (s *Store) Find(ctx context.Context, match func(Order) bool) (Order, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, customer, total FROM orders ORDER BY id`)
	if err != nil {
		return Order{}, fmt.Errorf("query orders: %w", err)
	}
	for rows.Next() {
		var o Order
		if err := rows.Scan(&o.ID, &o.Customer, &o.Total); err != nil {
			return Order{}, fmt.Errorf("scan order: %w", err)
		}
		if match(o) {
			return o, nil
		}
	}
	if err := rows.Err(); err != nil {
		return Order{}, fmt.Errorf("read orders: %w", err)
	}
	return Order{}, ErrNotFound
}

// ANCHOR_END: find
