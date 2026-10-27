package wallet_test

import (
	"errors"
	"fmt"

	wallet "github.com/holmeslin/learning-go-to-senior/examples/chapter2/01_sentinel_custom_opaque/before"
)

// ANCHOR: example

// shortfall 從錯誤訊息裡取出缺少的金額。
func shortfall(err error) (int, bool) {
	var need, have int
	_, scanErr := fmt.Sscanf(err.Error(), "insufficient funds: need %d, have %d", &need, &have)
	return need - have, scanErr == nil
}

// decide 是呼叫端的決定。
func decide(err error) string {
	switch {
	case errors.Is(err, wallet.ErrNotFound):
		return "404"
	case errors.Is(err, wallet.ErrInsufficientFunds):
		if n, ok := shortfall(err); ok {
			return fmt.Sprintf("422 還差 %d 元", n)
		}
		return "422 餘額不足"
	case errors.Is(err, wallet.ErrClosed), errors.Is(err, wallet.ErrInvalidAmount):
		return "500"
	default:
		return "500"
	}
}

func ExampleWallet_Withdraw() {
	w := wallet.New(map[string]int{"alice": 30})
	try := func(id string, amount int) {
		err := w.Withdraw(id, amount)
		fmt.Printf("%v -> %s\n", err, decide(err))
	}

	try("bob", 10)
	try("alice", 100)
	w.Close()
	try("alice", 10)
	// Output:
	// account not found -> 404
	// insufficient funds: need 100, have 30 -> 422 還差 70 元
	// wallet closed -> 500
}

// ANCHOR_END: example
