package breaker_test

import (
	"fmt"
	"time"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter10/05_circuit_breaker/after/breaker"
)

// ANCHOR: example

func Example() {
	clk := &clock{t: time.Unix(0, 0)}
	b := breaker.New(breaker.Config{Threshold: 2, Cooldown: 10 * time.Second, Now: clk.Now})

	fmt.Println("start:", b.State())
	b.Do(func() error { return errBoom })
	b.Do(func() error { return errBoom })
	fmt.Println("after 2 failures:", b.State())

	err := b.Do(func() error { return nil })
	fmt.Println("request while open:", err)

	clk.Advance(10 * time.Second)
	b.Do(func() error {
		fmt.Println("probe runs in state:", b.State())
		return nil
	})
	fmt.Println("after the probe succeeded:", b.State())
	// Output:
	// start: closed
	// after 2 failures: open
	// request while open: breaker: open
	// probe runs in state: half-open
	// after the probe succeeded: closed
}

// ANCHOR_END: example
