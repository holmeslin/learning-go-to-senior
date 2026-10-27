// Package wallet 管理帳戶餘額並提供提款。
//
// 三種失敗，呼叫端要做的決定不同，所以錯誤的種類也不同：
// ErrNotFound 是哨兵錯誤，InsufficientFundsError 是自訂型別，其餘都是不透明錯誤。
package wallet

import (
	"errors"
	"fmt"
	"maps"
)

// ANCHOR: errors

// ErrNotFound 表示帳戶不存在。呼叫端用 errors.Is 判斷。
var ErrNotFound = errors.New("account not found")

// InsufficientFundsError 表示餘額不足，並帶著呼叫端需要的數字。
// 呼叫端用 errors.AsType 取得。
type InsufficientFundsError struct {
	Need, Have int
}

// Error 實作 error。
func (e *InsufficientFundsError) Error() string {
	return fmt.Sprintf("insufficient funds: need %d, have %d", e.Need, e.Have)
}

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

// Withdraw 從帳戶 id 提出 amount 元，amount 必須大於 0。
//
// 帳戶不存在時回傳 ErrNotFound，餘額不足時回傳 *InsufficientFundsError；
// 其他失敗（Wallet 已關閉、amount 不合法）只保證 error 不是 nil。
func (w *Wallet) Withdraw(id string, amount int) error {
	if w.closed {
		return errors.New("wallet closed")
	}
	if amount <= 0 {
		return fmt.Errorf("withdraw %d: amount must be positive", amount)
	}
	have, ok := w.balances[id]
	if !ok {
		return ErrNotFound
	}
	if have < amount {
		return &InsufficientFundsError{Need: amount, Have: have}
	}
	w.balances[id] = have - amount
	return nil
}

// ANCHOR_END: withdraw
