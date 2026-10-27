// Package order 處理訂單。它不知道通知怎麼送，只定義自己需要什麼。
package order

import "fmt"

// Order 是一筆訂單。
type Order struct {
	ID     int
	Item   string
	Amount int
}

// ANCHOR: notifier

// Notifier 是 order 對「通知」的需求：訂單成立後，把它交出去。
// 介面由使用它的 order 定義，所以 order 不必 import 任何實作。
type Notifier interface {
	Notify(o Order) error
}

// ANCHOR_END: notifier

// ANCHOR: place

// Place 成立一筆訂單並通知。通知失敗時，仍回傳已成立的訂單與錯誤。
func Place(n Notifier, id int, item string, amount int) (Order, error) {
	o := Order{ID: id, Item: item, Amount: amount}
	if err := n.Notify(o); err != nil {
		return o, fmt.Errorf("order: notify: %w", err)
	}
	return o, nil
}

// ANCHOR_END: place
