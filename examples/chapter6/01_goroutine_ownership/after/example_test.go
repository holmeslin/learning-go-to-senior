package notify_test

import (
	"context"
	"errors"
	"fmt"
	"sync"

	notify "github.com/holmeslin/learning-go-to-senior/examples/chapter6/01_goroutine_ownership/after"
)

// ANCHOR: run

func ExampleNotifier_Run() {
	sent := make(chan string)
	n := notify.New(func(to string) error {
		sent <- to
		return nil
	})

	ctx, stop := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Go(func() { n.Run(ctx) }) // 擁有者開始 goroutine

	n.Welcome("a@example.com")
	n.Welcome("b@example.com")
	fmt.Println(<-sent)
	fmt.Println(<-sent)

	stop()
	wg.Wait() // 也由擁有者等它結束
	fmt.Println("stopped")
	// Output:
	// a@example.com
	// b@example.com
	// stopped
}

// ANCHOR_END: run

// ANCHOR: broadcast

func ExampleNotifier_Broadcast() {
	n := notify.New(func(to string) error {
		if to == "bad@example.com" {
			return errors.New("mailbox full")
		}
		return nil
	})

	failed := n.Broadcast([]string{"a@example.com", "bad@example.com", "c@example.com"})
	fmt.Println("failed:", failed)
	// Output:
	// failed: [bad@example.com]
}

// ANCHOR_END: broadcast
