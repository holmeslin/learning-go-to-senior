package coupon_test

import (
	"errors"
	"testing"

	coupon "github.com/holmeslin/learning-go-to-senior/examples/chapter5/02_table_driven_boundary/before"
)

func newBook() *coupon.Book {
	return coupon.NewBook(
		coupon.Coupon{Code: "SAVE10", Percent: 10},
		coupon.Coupon{Code: "WELCOME20", Percent: 20, MinSpend: 500, SingleUse: true},
	)
}

// ANCHOR: table

func TestRedeem(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(b *coupon.Book)
		code       string
		amount     int
		want       int
		wantErr    error
		skipAmount bool
		checkUsed  bool
		wantUsed   int
	}{
		{name: "九折", code: "SAVE10", amount: 1000, want: 900},
		{name: "折扣無條件捨去", code: "SAVE10", amount: 995, want: 896},
		{name: "代碼不分大小寫", code: "save10", amount: 1000, want: 900},
		{name: "剛好達到低消", code: "WELCOME20", amount: 500, want: 400},
		{name: "未知代碼", code: "NOPE", amount: 1000, wantErr: coupon.ErrUnknownCode},
		{name: "低於低消", code: "WELCOME20", amount: 499, wantErr: coupon.ErrMinSpend},
		{
			name:    "單次券用過就不能再用",
			setup:   func(b *coupon.Book) { _, _ = b.Redeem("WELCOME20", 500) },
			code:    "WELCOME20",
			amount:  500,
			wantErr: coupon.ErrUsedUp,
		},
		{name: "低消不足不消耗單次券", code: "WELCOME20", amount: 499, wantErr: coupon.ErrMinSpend, wantUsed: 0},
		{name: "兌換成功記一次使用", code: "SAVE10", amount: 1000, skipAmount: true, checkUsed: true, wantUsed: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := newBook()
			if tt.setup != nil {
				tt.setup(b)
			}
			got, err := b.Redeem(tt.code, tt.amount)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Redeem(%q, %d) error = %v, want %v", tt.code, tt.amount, err, tt.wantErr)
				}
			} else if err != nil {
				t.Fatalf("Redeem(%q, %d) error = %v, want nil", tt.code, tt.amount, err)
			} else if !tt.skipAmount && got != tt.want {
				t.Errorf("Redeem(%q, %d) = %d, want %d", tt.code, tt.amount, got, tt.want)
			}
			if tt.checkUsed && b.Used(tt.code) != tt.wantUsed {
				t.Errorf("Used(%q) = %d, want %d", tt.code, b.Used(tt.code), tt.wantUsed)
			}
		})
	}
}

// ANCHOR_END: table
