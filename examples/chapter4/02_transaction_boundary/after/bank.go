// Package bank 由用例決定交易範圍：Store 綁在 DB 或 Tx 上，自己不開交易。
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

// ANCHOR: store

// Execer 是 Store 需要的能力，*sql.DB 與 *sql.Tx 都符合。
type Execer interface {
	// ExecContext 執行不回傳資料列的 SQL。
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// Store 讀寫帳戶資料。它用傳進來的 Execer 執行 SQL，交易由呼叫端決定。
type Store struct {
	q Execer
}

// NewStore 回傳使用 q 的 Store；q 可以是 *sql.DB，也可以是 *sql.Tx。
func NewStore(q Execer) *Store {
	return &Store{q: q}
}

// ANCHOR_END: store

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

// apply 改餘額並記帳本，但不開交易：這兩件事是否一起成功，由呼叫端的交易決定。
func (s *Store) apply(ctx context.Context, name string, delta int) error {
	res, err := s.q.ExecContext(ctx, `UPDATE accounts SET balance = balance + ? WHERE name = ?`, delta, name)
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
	_, err = s.q.ExecContext(ctx, `INSERT INTO entries (account, delta) VALUES (?, ?)`, name, delta)
	return err
}

// ANCHOR_END: apply

// Service 是轉帳這個用例。它持有 *sql.DB，因為開交易是它的工作。
type Service struct {
	db *sql.DB
}

// NewService 回傳使用 db 的 Service。
func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

// ANCHOR: transfer

// Transfer 把 amount 從 from 轉到 to：先扣款，再入帳，兩步在同一個交易裡。
// 任何一步失敗，整個交易回滾。
func (s *Service) Transfer(ctx context.Context, from, to string, amount int) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transfer: %w", err)
	}
	defer tx.Rollback()

	store := NewStore(tx)
	if err = store.Debit(ctx, from, amount); err != nil {
		return err
	}
	if err = store.Credit(ctx, to, amount); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit transfer: %w", err)
	}
	return nil
}

// ANCHOR_END: transfer
