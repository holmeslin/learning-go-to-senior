// Package orders 是訂單的 store。每個查詢自己設一個固定的逾時，不看呼叫端的 context。
package orders

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// ANCHOR: store

// Store 讀取訂單。每個查詢自己設一個固定的逾時，免得卡住的查詢一直占著資料庫。
type Store struct {
	db      *sql.DB
	timeout time.Duration
}

// NewStore 回傳使用 db 的 Store；timeout 是每個查詢的逾時上限。
func NewStore(db *sql.DB, timeout time.Duration) *Store {
	return &Store{db: db, timeout: timeout}
}

// PendingCount 回傳待處理的訂單數。
func (s *Store) PendingCount(ctx context.Context) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout) // 收到的 ctx 在這裡被蓋掉了
	defer cancel()

	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM orders WHERE status = 'pending'`).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count pending orders: %w", err)
	}
	return n, nil
}

// ANCHOR_END: store
