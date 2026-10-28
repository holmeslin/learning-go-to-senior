package wallet_test

import (
	"errors"
	"fmt"

	wallet "github.com/holmeslin/learning-go-to-senior/examples/chapter2/01_sentinel_custom_opaque/after"
)

// ANCHOR: example

// decide 是呼叫端的決定：查無帳戶回 404，餘額不足告訴使用者差多少，其他一律 500。
func decide(err error) string {
	if errors.Is(err, wallet.ErrNotFound) {
		return "404"
	}
	if e, ok := errors.AsType[*wallet.InsufficientFundsError](err); ok {
		return fmt.Sprintf("422 還差 %d 元", e.Need-e.Have)
	}
	return "500"
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
