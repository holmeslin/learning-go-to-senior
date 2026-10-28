// Package bank 是常見寫法：每個 store 方法各自開交易，轉帳由用例依序呼叫它們。
package bank

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Schema 建立範例需要的資料表：帳戶餘額，以及每次餘額異動的帳本。
const Schema = `
CREATE TABLE accounts (name TEXT PRIMARY KEY, balance INTEGER NOT NULL CHECK (balance >= 0));
CREATE TABLE entries (account TEXT NOT NULL, delta INTEGER NOT NULL);
`

// ErrNotFound 表示帳戶不存在。
var ErrNotFound = errors.New("account not found")

// Store 讀寫帳戶資料。
type Store struct {
	db *sql.DB
}

// NewStore 回傳使用 db 的 Store。
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// Debit 從 name 扣掉 amount，並記一筆帳本。
func (s *Store) Debit(ctx context.Context, name string, amount int) error {
	if err := s.apply(ctx, name, -amount); err != nil {
		return fmt.Errorf("debit %s: %w", name, err)
	}
	return nil
}

// Credit 給 name 加上 amount，並記一筆帳本。帳戶不存在時回傳 ErrNotFound。
func (s *Store) Credit(ctx context.Context, name string, amount int) error {
	if err := s.apply(ctx, name, amount); err != nil {
		return fmt.Errorf("credit %s: %w", name, err)
	}
	return nil
}

// ANCHOR: apply

// apply 在自己開的交易裡改餘額並記帳本：這兩件事要一起成功。
func (s *Store) apply(ctx context.Context, name string, delta int) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, `UPDATE accounts SET balance = balance + ? WHERE name = ?`, delta, name)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO entries (account, delta) VALUES (?, ?)`, name, delta); err != nil {
		return err
	}
	return tx.Commit()
}

// ANCHOR_END: apply

// Service 是轉帳這個用例。
type Service struct {
	store *Store
}

// NewService 回傳使用 store 的 Service。
func NewService(store *Store) *Service {
	return &Service{store: store}
}

// ANCHOR: transfer

// Transfer 把 amount 從 from 轉到 to：先扣款，再入帳。
func (s *Service) Transfer(ctx context.Context, from, to string, amount int) error {
	if err := s.store.Debit(ctx, from, amount); err != nil {
		return err
	}
	return s.store.Credit(ctx, to, amount)
}

// ANCHOR_END: transfer
