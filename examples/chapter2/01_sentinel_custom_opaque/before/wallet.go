// Package wallet 管理帳戶餘額並提供提款。
// 這是「每一種失敗都定義一個哨兵錯誤」的版本。
package wallet

import (
	"errors"
	"fmt"
	"maps"
)

// ANCHOR: errors

var (
	// ErrNotFound 表示帳戶不存在。
	ErrNotFound = errors.New("account not found")
	// ErrInsufficientFunds 表示餘額不足。
	ErrInsufficientFunds = errors.New("insufficient funds")
	// ErrInvalidAmount 表示金額不合法。
	ErrInvalidAmount = errors.New("invalid amount")
	// ErrClosed 表示 Wallet 已經關閉。
	ErrClosed = errors.New("wallet closed")
)

// ANCHOR_END: errors

// Wallet 保存各帳戶的餘額。
type Wallet struct {
	balances map[string]int
	closed   bool
}

// New 回傳一個 Wallet，初始餘額複製自 balances。
func New(balances map[string]int) *Wallet {
	return &Wallet{balances: maps.Clone(balances)}
}

// Close 關閉 Wallet，之後的 Withdraw 都會失敗。
func (w *Wallet) Close() { w.closed = true }

// ANCHOR: withdraw

// Withdraw 從帳戶 id 提出 amount 元。
func (w *Wallet) Withdraw(id string, amount int) error {
	if w.closed {
		return ErrClosed
	}
	if amount <= 0 {
		return ErrInvalidAmount
	}
	have, ok := w.balances[id]
	if !ok {
		return ErrNotFound
	}
	if have < amount {
		return fmt.Errorf("%w: need %d, have %d", ErrInsufficientFunds, amount, have)
	}
	w.balances[id] = have - amount
	return nil
}

// ANCHOR_END: withdraw
