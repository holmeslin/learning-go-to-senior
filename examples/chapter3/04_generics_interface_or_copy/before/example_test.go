package ledger_test

import (
	"fmt"

	ledger "github.com/holmeslin/learning-go-to-senior/examples/chapter3/04_generics_interface_or_copy/before"
)

// ANCHOR: example

func ExampleTotal() {
	orders := []ledger.Order{
		{ID: "o1", Cents: 1200},
		{ID: "o2", Cents: 800, Cancelled: true},
		{ID: "o3", Cents: 500},
	}
	refunds := []ledger.Refund{
		{ID: "r1", Cents: 300, Approved: true},
		{ID: "r2", Cents: 200},
	}
	fmt.Println(ledger.Total(orders))
	fmt.Println(ledger.Total(refunds))

	// 別處傳來的 []int 與 []float64 不能直接轉型成 []ledger.Cents、[]ledger.GB，
	// 只能一個一個包起來。
	cents := []int{1200, 500, 300}
	wrapped := make([]ledger.Cents, len(cents))
	for i, c := range cents {
		wrapped[i] = ledger.Cents(c)
	}
	fmt.Println(ledger.Total(wrapped))

	readings := []float64{0.5, 1.25, 2}
	usage := make([]ledger.GB, len(readings))
	for i, r := range readings {
		usage[i] = ledger.GB(r)
	}
	fmt.Println(ledger.Total(usage))
	// Output:
	// 1700
	// 300
	// 2000
	// 3.75
}

// ANCHOR_END: example
