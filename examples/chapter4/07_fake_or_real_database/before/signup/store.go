package signup

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
)

// Schema 是 users 資料表的 SQL，部署與測試用的是同一份。
//
//go:embed schema.sql
var Schema string

// SQLStore 把使用者存進資料庫。
type SQLStore struct {
	db *sql.DB
}

// NewSQLStore 回傳使用 db 的 SQLStore；資料表須已用 Schema 建好。
func NewSQLStore(db *sql.DB) *SQLStore {
	return &SQLStore{db: db}
}

// ANCHOR: add

// Add 新增一個信箱，實作 Store。
func (s *SQLStore) Add(ctx context.Context, email string) (User, error) {
	res, err := s.db.ExecContext(ctx, `INSERT INTO users (email) VALUES (?)`, email)
	if err != nil {
		return User{}, fmt.Errorf("insert %s: %w", email, err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return User{}, fmt.Errorf("last insert id: %w", err)
	}
	return User{ID: id, Email: email}, nil
}

// ANCHOR_END: add
