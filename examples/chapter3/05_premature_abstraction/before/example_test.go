package amount_test

import (
	"fmt"

	amount "github.com/holmeslin/learning-go-to-senior/examples/chapter3/05_premature_abstraction/before"
)

// ANCHOR: callers

func ExampleFormat() {
	const refund = -1200

	fmt.Println(amount.Format(refund, false, false, false)) // CSV 匯出
	fmt.Println(amount.Format(refund, true, true, false))   // 收據
	fmt.Println(amount.Format(refund, false, true, true))   // 帳務報表
	// Output:
	// -12.00
	// -$12.00
	// (12.00)
}

// ANCHOR_END: callers

// ANCHOR: invoice

func ExampleFormat_invoice() {
	for _, cents := range []int64{123450, -1200, 0} {
		fmt.Println(amount.Format(cents, true, true, true)) // 發票
	}
	// Output:
	// $1,234.50
	// ($12.00)
	// $-
}

// ANCHOR_END: invoice
