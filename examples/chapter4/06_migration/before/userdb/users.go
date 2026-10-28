package userdb

import (
	"context"
	"database/sql"
)

// ANCHOR: users

// 第二版：users 多了 name，直接改這句 CREATE TABLE。
func ensureUsers(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS users (
	id    INTEGER PRIMARY KEY,
	email TEXT NOT NULL UNIQUE,
	name  TEXT NOT NULL DEFAULT ''
)`)
	return err
}

// ANCHOR_END: users

// AddUser 新增一位使用者。
func AddUser(ctx context.Context, db *sql.DB, email, name string) error {
	_, err := db.ExecContext(ctx, `INSERT INTO users (email, name) VALUES (?, ?)`, email, name)
	return err
}
