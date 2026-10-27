package wallet_test

import (
	"errors"
	"testing"

	wallet "github.com/holmeslin/learning-go-to-senior/examples/chapter2/01_sentinel_custom_opaque/after"
)

func TestWithdrawSucceeds(t *testing.T) {
	w := wallet.New(map[string]int{"alice": 30})
	if err := w.Withdraw("alice", 30); err != nil {
		t.Fatalf("Withdraw: %v", err)
	}
	// 餘額已經扣光，再提 1 元會餘額不足。
	if err := w.Withdraw("alice", 1); err == nil {
		t.Fatal("second Withdraw succeeded; want insufficient funds")
	}
}

// 呼叫端需要分辨「查無帳戶」：哨兵錯誤用 errors.Is。
func TestWithdrawNotFound(t *testing.T) {
	w := wallet.New(nil)
	err := w.Withdraw("bob", 10)
	if !errors.Is(err, wallet.ErrNotFound) {
		t.Fatalf("err = %v; want ErrNotFound", err)
	}
}

// 呼叫端需要缺少的金額：自訂型別用 errors.AsType 取出欄位。
func TestWithdrawInsufficientFunds(t *testing.T) {
	w := wallet.New(map[string]int{"alice": 30})
	err := w.Withdraw("alice", 100)
	e, ok := errors.AsType[*wallet.InsufficientFundsError](err)
	if !ok {
		t.Fatalf("err = %v; want *InsufficientFundsError", err)
	}
	if e.Need != 100 || e.Have != 30 {
		t.Errorf("Need, Have = %d, %d; want 100, 30", e.Need, e.Have)
	}
	if errors.Is(err, wallet.ErrNotFound) {
		t.Error("insufficient funds error matches ErrNotFound")
	}
}

// 不透明錯誤：呼叫端只知道失敗了，沒有任何東西可以拿來分辨。
func TestOpaqueErrors(t *testing.T) {
	closed := wallet.New(map[string]int{"alice": 30})
	closed.Close()

	tests := []struct {
		name string
		w    *wallet.Wallet
		amt  int
	}{
		{"closed", closed, 10},
		{"zero amount", wallet.New(map[string]int{"alice": 30}), 0},
		{"negative amount", wallet.New(map[string]int{"alice": 30}), -5},
	}
	for _, tt := range tests {
		err := tt.w.Withdraw("alice", tt.amt)
		if err == nil {
			t.Errorf("%s: Withdraw succeeded; want error", tt.name)
			continue
		}
		if errors.Is(err, wallet.ErrNotFound) {
			t.Errorf("%s: err matches ErrNotFound", tt.name)
		}
		if _, ok := errors.AsType[*wallet.InsufficientFundsError](err); ok {
			t.Errorf("%s: err is *InsufficientFundsError", tt.name)
		}
	}
}
