package notify_test

import (
	"fmt"
	"slices"

	notify "github.com/holmeslin/learning-go-to-senior/examples/chapter6/01_goroutine_ownership/before"
)

// ANCHOR: close

func ExampleNotifier_Close() {
	// release 讓寄信卡住，只是為了讓下面的觀察每次都一樣；Notifier 本身沒有辦法問「寄完了沒」。
	release := make(chan struct{})
	delivered := make(chan string, 1)
	n := notify.New(func(to string) error {
		<-release
		delivered <- to
		return nil
	})

	n.Welcome("a@example.com")
	n.Close()
	fmt.Println("Close returned, delivered:", len(delivered))

	close(release)
	fmt.Println("later:", <-delivered)
	// Output:
	// Close returned, delivered: 0
	// later: a@example.com
}

// ANCHOR_END: close

// ANCHOR: broadcast

func ExampleNotifier_Broadcast() {
	release := make(chan struct{})
	delivered := make(chan string, 3)
	n := notify.New(func(to string) error {
		<-release
		delivered <- to
		return nil
	})

	n.Broadcast([]string{"a@example.com", "b@example.com", "c@example.com"})
	fmt.Println("Broadcast returned, delivered:", len(delivered))

	close(release)
	got := []string{<-delivered, <-delivered, <-delivered}
	slices.Sort(got)
	fmt.Println("later:", got)
	n.Close()
	// Output:
	// Broadcast returned, delivered: 0
	// later: [a@example.com b@example.com c@example.com]
}

// ANCHOR_END: broadcast
