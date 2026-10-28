// Package orders 的 store 不自己決定查詢的逾時：逾時與取消由呼叫端的 context 決定。
package orders

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Store 讀取訂單。
type Store struct {
	db          *sql.DB
	searchLimit time.Duration
}

// NewStore 回傳使用 db 的 Store。searchLimit 是 SearchCount 自己加上的上限。
func NewStore(db *sql.DB, searchLimit time.Duration) *Store {
	return &Store{db: db, searchLimit: searchLimit}
}

// ANCHOR: store

// PendingCount 回傳待處理的訂單數。逾時與取消完全由 ctx 決定。
func (s *Store) PendingCount(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM orders WHERE status = 'pending'`).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count pending orders: %w", err)
	}
	return n, nil
}

// ANCHOR_END: store

// ANCHOR: search

// SearchCount 回傳備註含有 keyword 的訂單數。備註沒有索引，查詢會掃過整張表，
// 所以在收到的 ctx 上再縮短一層上限；ctx 的取消與更早的 deadline 照樣有效。
func (s *Store) SearchCount(ctx context.Context, keyword string) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, s.searchLimit)
	defer cancel()

	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM orders WHERE instr(note, ?) > 0`, keyword).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("search orders: %w", err)
	}
	return n, nil
}

// ANCHOR_END: search
