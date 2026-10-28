package shophttp_test

import (
	"fmt"
	"net/http"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter4/01_repository_layer/after/order"
	"github.com/holmeslin/learning-go-to-senior/examples/chapter4/01_repository_layer/after/shophttp"
)

// ANCHOR: example

func ExampleNewHandler_closeAccount() {
	db := mustOpenDB("ExampleNewHandler_closeAccount")
	defer db.Close()
	seed(db, 1, "keyboard", "paid") // 已付款、還沒出貨
	h := shophttp.NewHandler(order.NewStore(db))

	fmt.Println("open orders:", call(h, http.MethodGet, "/users/1/orders/open"))
	fmt.Println("close account:", call(h, http.MethodPost, "/users/1/close"))
	// Output:
	// open orders: 200 1 keyboard paid
	// close account: 409 user has open orders
}

// ANCHOR_END: example
