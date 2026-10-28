// Package orders 查詢訂單，並在 Open 裡刻意設定連線池。
package orders

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
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

// ANCHOR: open

// Open 開啟資料庫並設定連線池。maxOpen 是這個行程最多能占用的連線數，
// 由呼叫端依資料庫的連線上限與服務的副本數決定，必須至少是 1：
// database/sql 把 0 與負數當成「不設上限」。
func Open(dsn string, maxOpen int) (*sql.DB, error) {
	if maxOpen < 1 {
		return nil, fmt.Errorf("max open connections must be at least 1, got %d", maxOpen)
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxOpen)             // 預設只留 2 條；尖峰過後不必每次重新連線
	db.SetConnMaxLifetime(30 * time.Minute) // 定期換新，避開中間設備默默切斷的老連線
	return db, nil
}

// ANCHOR_END: open

// ANCHOR: find

// Find 依 id 順序找出第一筆讓 match 回傳 true 的訂單；沒有符合的回傳 ErrNotFound。
func (s *Store) Find(ctx context.Context, match func(Order) bool) (Order, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, customer, total FROM orders ORDER BY id`)
	if err != nil {
		return Order{}, fmt.Errorf("query orders: %w", err)
	}
	defer rows.Close()
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
