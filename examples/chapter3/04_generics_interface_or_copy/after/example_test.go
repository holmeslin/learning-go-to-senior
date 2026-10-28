package ledger_test

import (
	"fmt"
	"time"

	ledger "github.com/holmeslin/learning-go-to-senior/examples/chapter3/04_generics_interface_or_copy/after"
)

// ANCHOR: example

func Example() {
	orders := []ledger.Order{
		{ID: "o1", Cents: 1200},
		{ID: "o2", Cents: 800, Cancelled: true},
		{ID: "o3", Cents: 500},
	}
	refunds := []ledger.Refund{
		{ID: "r1", Cents: 300, Approved: true},
		{ID: "r2", Cents: 200},
	}
	fmt.Println(ledger.OrderTotal(orders))
	fmt.Println(ledger.RefundTotal(refunds))

	// 別處傳來的 []int 與 []float64，直接傳進去就好。
	fmt.Println(ledger.Sum([]int{1200, 500, 300}))
	fmt.Println(ledger.Sum([]float64{0.5, 1.25, 2}))
	// Output:
	// 1700
	// 300
	// 2000
	// 3.75
}

// ANCHOR_END: example

// ANCHOR: duration

func ExampleSum_duration() {
	fmt.Println(ledger.Sum([]time.Duration{time.Second, 1500 * time.Millisecond}))
	// Output: 2.5s
}

// ANCHOR_END: duration
