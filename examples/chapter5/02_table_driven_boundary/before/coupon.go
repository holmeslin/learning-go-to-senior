// Package coupon 處理折價券的兌換：依折扣百分比算出應付金額，並限制低消與使用次數。
package coupon

import (
	"errors"
	"fmt"
	"strings"
	"sync"
)

var (
	// ErrUnknownCode 表示沒有這個折價券代碼。
	ErrUnknownCode = errors.New("unknown code")
	// ErrMinSpend 表示消費金額低於這張券的低消。
	ErrMinSpend = errors.New("below minimum spend")
	// ErrUsedUp 表示單次券已經兌換過了。
	ErrUsedUp = errors.New("used up")
)

// Coupon 是一張折價券的規則。
type Coupon struct {
	Code      string // 代碼，兌換時不分大小寫
	Percent   int    // 折扣百分比，10 代表打九折
	MinSpend  int    // 低消；0 表示沒有低消
	SingleUse bool   // 只能兌換一次
}

// Book 記錄可用的折價券與各張券已兌換的次數，可以同時被多個 goroutine 使用。
type Book struct {
	mu      sync.Mutex
	coupons map[string]Coupon
	used    map[string]int
}

// NewBook 建立一個 Book，裡面已經有 coupons 這幾張券。
func NewBook(coupons ...Coupon) *Book {
	b := &Book{coupons: map[string]Coupon{}, used: map[string]int{}}
	for _, c := range coupons {
		b.coupons[strings.ToUpper(c.Code)] = c
	}
	return b
}

// ANCHOR: redeem

// Redeem 兌換折價券 code，回傳 amount 折扣後的應付金額（單位：分）。
// 折扣金額無條件捨去到整數；被拒絕的兌換不算使用一次。
func (b *Book) Redeem(code string, amount int) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	key := strings.ToUpper(code)
	c, ok := b.coupons[key]
	if !ok {
		return 0, fmt.Errorf("redeem %s: %w", code, ErrUnknownCode)
	}
	if c.SingleUse && b.used[key] > 0 {
		return 0, fmt.Errorf("redeem %s: %w", code, ErrUsedUp)
	}
	if amount < c.MinSpend {
		return 0, fmt.Errorf("redeem %s: %w of %d", code, ErrMinSpend, c.MinSpend)
	}
	b.used[key]++
	return amount - amount*c.Percent/100, nil
}

// ANCHOR_END: redeem

// Used 回傳 code 已經兌換成功的次數。
func (b *Book) Used(code string) int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.used[strings.ToUpper(code)]
}
