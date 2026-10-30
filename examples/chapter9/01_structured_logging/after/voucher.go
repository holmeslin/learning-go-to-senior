// Package voucher 核銷折價券；日誌寫進呼叫端傳入的 logger。
package voucher

import (
	"context"
	"errors"
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

// ANCHOR: logvalue

// LogValue 讓 slog 把 Voucher 記成 code 與 cents 兩個欄位。
func (v Voucher) LogValue() slog.Value {
	return slog.GroupValue(slog.String("code", v.Code), slog.Int64("cents", v.Cents))
}

// ANCHOR_END: logvalue

// Redeemer 保存券並記錄哪些券已經核銷過。
type Redeemer struct {
	log      *slog.Logger
	mu       sync.Mutex
	vouchers map[string]Voucher
	used     map[string]bool
}

// New 建立 Redeemer，日誌寫進 log；log 是 nil 時丟棄日誌，不會去用全域的 logger。
func New(log *slog.Logger, vouchers ...Voucher) *Redeemer {
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	r := &Redeemer{log: log, vouchers: map[string]Voucher{}, used: map[string]bool{}}
	for _, v := range vouchers {
		r.vouchers[v.Code] = v
	}
	return r
}

// ANCHOR: redeem

// Redeem 讓 userID 核銷 code 這張券，結果以欄位寫進 New 傳入的 logger。
func (r *Redeemer) Redeem(ctx context.Context, userID, code string) error {
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
		r.log.LogAttrs(ctx, slog.LevelWarn, "redeem failed",
			slog.String("user_id", userID), slog.String("code", code), slog.Any("error", err))
		return err
	}
	r.log.LogAttrs(ctx, slog.LevelInfo, "voucher redeemed",
		slog.String("user_id", userID), slog.Any("voucher", v))
	return nil
}

// ANCHOR_END: redeem
