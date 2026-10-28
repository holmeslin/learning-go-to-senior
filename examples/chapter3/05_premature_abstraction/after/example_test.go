package amount_test

import (
	"fmt"

	amount "github.com/holmeslin/learning-go-to-senior/examples/chapter3/05_premature_abstraction/after"
)

// ANCHOR: example

func Example() {
	const refund = -1200

	fmt.Println(amount.CSV(refund))     // CSV 匯出
	fmt.Println(amount.Receipt(refund)) // 收據
	fmt.Println(amount.Ledger(refund))  // 帳務報表
	for _, cents := range []int64{123450, refund, 0} {
		fmt.Println(amount.Invoice(cents)) // 發票
	}
	// Output:
	// -12.00
	// -$12.00
	// (12.00)
	// $1,234.50
	// ($12.00)
	// $0.00
}

// ANCHOR_END: example
