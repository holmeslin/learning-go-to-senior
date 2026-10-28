package bank_test

import (
	"context"
	"fmt"

	bank "github.com/holmeslin/learning-go-to-senior/examples/chapter4/02_transaction_boundary/after"
)

// ANCHOR: example

func ExampleService_Transfer() {
	db := newDB()
	defer db.Close()
	svc := bank.NewService(db)

	// nobody 這個帳戶不存在：入帳失敗，連已經執行的扣款一起回滾。
	err := svc.Transfer(context.Background(), "ann", "nobody", 300)
	balances, ledger := readState(db)

	fmt.Println("transfer:", err)
	fmt.Println("balances:", balances)
	fmt.Println("ledger:", ledger)
	// Output:
	// transfer: credit nobody: account not found
	// balances: map[ann:500 bob:100]
	// ledger: []
}

func ExampleService_Transfer_commit() {
	db := newDB()
	defer db.Close()
	svc := bank.NewService(db)

	err := svc.Transfer(context.Background(), "ann", "bob", 300)
	balances, ledger := readState(db)

	fmt.Println("transfer:", err)
	fmt.Println("balances:", balances)
	fmt.Println("ledger:", ledger)
	// Output:
	// transfer: <nil>
	// balances: map[ann:200 bob:400]
	// ledger: [ann:-300 bob:+300]
}

// ANCHOR_END: example
