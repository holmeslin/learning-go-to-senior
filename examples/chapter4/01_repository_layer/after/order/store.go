// Package order 保存訂單。所有 SQL 都在這個套件裡，呼叫端只看到領域型別與領域錯誤。
package order

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// ErrNotFound 表示沒有符合的訂單。用 errors.Is 判斷。
var ErrNotFound = errors.New("order not found")

// Order 是一筆訂單。Status 是 "pending"、"paid" 或 "shipped"。
type Order struct {
	ID     int64
	UserID int64
	Item   string
	Status string
}

// Store 把訂單存在資料庫裡。
type Store struct {
	db *sql.DB
}

// NewStore 回傳使用 db 的 Store。db 由呼叫端開啟與關閉，orders 資料表也要先建好。
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// ANCHOR: find

// Find 回傳編號為 id 的訂單。沒有這筆訂單時，回傳的錯誤符合 ErrNotFound。
// 其他錯誤（包含 ctx 被取消或逾時）原樣包裝，errors.Is 找得到它們。
func (s *Store) Find(ctx context.Context, id int64) (Order, error) {
	o := Order{ID: id}
	err := s.db.QueryRowContext(ctx,
		`SELECT user_id, item, status FROM orders WHERE id = ?`, id,
	).Scan(&o.UserID, &o.Item, &o.Status)
	if errors.Is(err, sql.ErrNoRows) {
		return Order{}, fmt.Errorf("find order %d: %w", id, ErrNotFound)
	}
	if err != nil {
		return Order{}, fmt.Errorf("find order %d: %w", id, err)
	}
	return o, nil
}

// ANCHOR_END: find

// ANCHOR: open

// OpenOrdersFor 回傳 userID 尚未出貨（pending 或 paid）的訂單，依編號排序。
// 沒有任何未完成的訂單時回傳空的結果，不是錯誤。
func (s *Store) OpenOrdersFor(ctx context.Context, userID int64) ([]Order, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, user_id, item, status FROM orders
		 WHERE user_id = ? AND status IN ('pending', 'paid') ORDER BY id`, userID)
	if err != nil {
		return nil, fmt.Errorf("list open orders of user %d: %w", userID, err)
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var o Order
		if err := rows.Scan(&o.ID, &o.UserID, &o.Item, &o.Status); err != nil {
			return nil, fmt.Errorf("scan open order of user %d: %w", userID, err)
		}
		orders = append(orders, o)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list open orders of user %d: %w", userID, err)
	}
	return orders, nil
}

// ANCHOR_END: open
