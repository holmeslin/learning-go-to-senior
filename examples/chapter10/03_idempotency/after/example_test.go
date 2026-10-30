package payment_test

import (
	"fmt"
	"strings"

	payment "github.com/holmeslin/learning-go-to-senior/examples/chapter10/03_idempotency/after"
)

// ANCHOR: example

func Example() {
	ledger := &payment.Ledger{}
	h := payment.NewStore().Wrap(payment.Handler(ledger))

	// 同一個動作的兩次嘗試用同一個鍵；第二次就是 client 逾時後的重試。
	for range 2 {
		rec := post(h, "pay-A-1", `{"order":"A-1","cents":1200}`)
		fmt.Printf("%d %s replayed=%q\n",
			rec.Code, strings.TrimSpace(rec.Body.String()), rec.Header().Get("Idempotent-Replayed"))
	}

	for _, c := range ledger.Charges() {
		fmt.Println("ledger:", c.ID, c.Order, c.Cents)
	}
	// Output:
	// 201 {"id":"ch_1","order":"A-1","cents":1200} replayed=""
	// 201 {"id":"ch_1","order":"A-1","cents":1200} replayed="true"
	// ledger: ch_1 A-1 1200
}

// ANCHOR_END: example
