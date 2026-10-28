package loyalty_test

import (
	"errors"
	"fmt"

	loyalty "github.com/holmeslin/learning-go-to-senior/examples/chapter3/03_redundant_interface/after"
)

// printNotifier 把通知印出來，代替真正的寄信。
type printNotifier struct{}

func (printNotifier) Notify(id, msg string) error {
	fmt.Printf("notify %s: %s\n", id, msg)
	return nil
}

// ANCHOR: example

func ExampleService_Earn() {
	store := loyalty.NewStore(loyalty.Member{ID: "alice", Points: 90})
	svc := loyalty.NewService(store, printNotifier{})

	points, err := svc.Earn("alice", 200)
	fmt.Println(points, err)

	// 沒有替身：找不到會員就是真的 Store 回傳的錯誤。
	_, err = svc.Earn("ghost", 200)
	fmt.Println(err, errors.Is(err, loyalty.ErrNotFound))
	// Output:
	// notify alice: gold member
	// 110 <nil>
	// earn: member ghost: not found true
}

// ANCHOR_END: example
