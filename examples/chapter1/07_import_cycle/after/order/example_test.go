package order_test

import (
	"fmt"
	"os"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter1/07_import_cycle/after/notify"
	"github.com/holmeslin/learning-go-to-senior/examples/chapter1/07_import_cycle/after/order"
)

// ANCHOR: example

func ExamplePlace() {
	// 接線在呼叫端：order 不認得 notify，是這裡把兩者接起來。
	n := notify.Console{W: os.Stdout}

	o, err := order.Place(n, 1, "鍵盤", 1200)
	if err != nil {
		fmt.Println("unexpected:", err)
		return
	}
	fmt.Println(o.Item)
	// Output:
	// 訂單 #1 已成立：鍵盤（1200 元）
	// 鍵盤
}

// ANCHOR_END: example
