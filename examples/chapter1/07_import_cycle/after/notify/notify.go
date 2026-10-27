// Package notify 把訂單通知寫到指定的輸出。
package notify

import (
	"fmt"
	"io"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter1/07_import_cycle/after/order"
)

// ANCHOR: console
// Console 把通知寫到 W。它依賴 order（要讀 Order 的欄位），反過來沒有。
type Console struct {
	W io.Writer
}

// 編譯期確認 Console 滿足 order.Notifier。
var _ order.Notifier = Console{}

// Notify 實作 order.Notifier。
func (c Console) Notify(o order.Order) error {
	_, err := fmt.Fprintf(c.W, "訂單 #%d 已成立：%s（%d 元）\n", o.ID, o.Item, o.Amount)
	return err
}

// ANCHOR_END: console
