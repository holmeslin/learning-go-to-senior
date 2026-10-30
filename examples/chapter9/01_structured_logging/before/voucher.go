// Package voucher 核銷折價券；每次核銷都寫一行日誌。
package voucher

import (
	"errors"
	"fmt"
	"log/slog"
	"sync"
)

// Voucher 是一張折價券。
type Voucher struct {
	// Code 是券號。
	Code string
	// Cents 是折抵的金額，單位是分。
	Cents int64
}

// Redeemer 保存券並記錄哪些券已經核銷過。
type Redeemer struct {
	mu       sync.Mutex
	vouchers map[string]Voucher
	used     map[string]bool
}

// New 建立 Redeemer，可核銷的券由 vouchers 給定。
func New(vouchers ...Voucher) *Redeemer {
	r := &Redeemer{vouchers: map[string]Voucher{}, used: map[string]bool{}}
	for _, v := range vouchers {
		r.vouchers[v.Code] = v
	}
	return r
}

// ANCHOR: redeem

// Redeem 讓 userID 核銷 code 這張券，結果寫進全域的 logger。
func (r *Redeemer) Redeem(userID, code string) error {
	r.mu.Lock()
	v, ok := r.vouchers[code]
	used := r.used[code]
	if ok && !used {
		r.used[code] = true
	}
	r.mu.Unlock()

	var err error
	switch {
	case !ok:
		err = errors.New("unknown voucher")
	case used:
		err = errors.New("voucher already redeemed")
	}
	if err != nil {
		slog.Warn(fmt.Sprintf("redeem failed: user %s code %s: %v", userID, code, err))
		return err
	}
	slog.Info(fmt.Sprintf("voucher redeemed: user %s code %s cents %d", userID, code, v.Cents))
	return nil
}

// ANCHOR_END: redeem
