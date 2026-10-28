package loyalty_test

import (
	"errors"
	"fmt"

	loyalty "github.com/holmeslin/learning-go-to-senior/examples/chapter3/03_redundant_interface/after"
)

// ANCHOR: example

func ExampleService_Earn() {
	n := &fakeNotifier{}
	store := loyalty.NewStore(loyalty.Member{ID: "alice", Points: 90})
	svc := loyalty.NewService(store, n)

	points, err := svc.Earn("alice", 200)
	fmt.Println(points, err, n.sent)

	// Store 沒有替身：找不到會員就是真的 Store 回傳的錯誤。
	_, err = svc.Earn("ghost", 200)
	fmt.Println(err, errors.Is(err, loyalty.ErrNotFound))
	// Output:
	// 110 <nil> [alice]
	// earn: member ghost: not found true
}

// ANCHOR_END: example
