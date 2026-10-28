package shop_test

import (
	"fmt"
	"net/http"

	shop "github.com/holmeslin/learning-go-to-senior/examples/chapter4/01_repository_layer/before"
)

// ANCHOR: example

func ExampleNewHandler_drift() {
	db := mustOpenDB("ExampleNewHandler_drift")
	defer db.Close()
	seed(db, 1, "keyboard", "paid") // 已付款、還沒出貨
	h := shop.NewHandler(db)

	fmt.Println("open orders:", call(h, http.MethodGet, "/users/1/orders/open"))
	fmt.Println("close account:", call(h, http.MethodPost, "/users/1/close"))
	// Output:
	// open orders: 200 1 keyboard paid
	// close account: 204
}

// ANCHOR_END: example
