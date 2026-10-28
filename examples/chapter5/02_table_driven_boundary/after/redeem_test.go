package coupon_test

import (
	"errors"
	"testing"

	coupon "github.com/holmeslin/learning-go-to-senior/examples/chapter5/02_table_driven_boundary/after"
)

// newBook 每次都建一本新的，測試之間不共用狀態。
func newBook() *coupon.Book {
	return coupon.NewBook(
		coupon.Coupon{Code: "SAVE10", Percent: 10},
		coupon.Coupon{Code: "WELCOME20", Percent: 20, MinSpend: 500, SingleUse: true},
	)
}

// ANCHOR: amount

func TestRedeemAmount(t *testing.T) {
	tests := []struct {
		name   string
		code   string
		amount int
		want   int
	}{
		{"九折", "SAVE10", 1000, 900},
		{"折扣無條件捨去", "SAVE10", 995, 896},
		{"代碼不分大小寫", "save10", 1000, 900},
		{"剛好達到低消", "WELCOME20", 500, 400},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newBook().Redeem(tt.code, tt.amount)
			if err != nil {
				t.Fatalf("Redeem(%q, %d) error = %v, want nil", tt.code, tt.amount, err)
			}
			if got != tt.want {
				t.Errorf("Redeem(%q, %d) = %d, want %d", tt.code, tt.amount, got, tt.want)
			}
		})
	}
}

// ANCHOR_END: amount

// ANCHOR: rejects

func TestRedeemRejects(t *testing.T) {
	tests := []struct {
		name   string
		code   string
		amount int
		want   error
	}{
		{"未知代碼", "NOPE", 1000, coupon.ErrUnknownCode},
		{"低於低消", "WELCOME20", 499, coupon.ErrMinSpend},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newBook().Redeem(tt.code, tt.amount)
			if !errors.Is(err, tt.want) {
				t.Errorf("Redeem(%q, %d) error = %v, want %v", tt.code, tt.amount, err, tt.want)
			}
		})
	}
}

// ANCHOR_END: rejects

// ANCHOR: stateful

func TestSingleUseCouponRedeemsOnce(t *testing.T) {
	b := newBook()
	if _, err := b.Redeem("WELCOME20", 500); err != nil {
		t.Fatalf("first Redeem error = %v, want nil", err)
	}

	_, err := b.Redeem("WELCOME20", 500)

	if !errors.Is(err, coupon.ErrUsedUp) {
		t.Errorf("second Redeem error = %v, want %v", err, coupon.ErrUsedUp)
	}
}

func TestRejectedRedeemKeepsSingleUseCoupon(t *testing.T) {
	b := newBook()
	if _, err := b.Redeem("WELCOME20", 499); !errors.Is(err, coupon.ErrMinSpend) {
		t.Fatalf("Redeem(499) error = %v, want %v", err, coupon.ErrMinSpend)
	}

	got, err := b.Redeem("WELCOME20", 500)

	if err != nil || got != 400 {
		t.Errorf("Redeem(500) after a rejection = %d, %v; want 400, nil", got, err)
	}
	if used := b.Used("WELCOME20"); used != 1 {
		t.Errorf("Used = %d, want 1", used)
	}
}

// ANCHOR_END: stateful

func TestRedeemCountsUse(t *testing.T) {
	b := newBook()
	if _, err := b.Redeem("SAVE10", 1000); err != nil {
		t.Fatal(err)
	}
	if used := b.Used("SAVE10"); used != 1 {
		t.Errorf("Used = %d, want 1", used)
	}
}
