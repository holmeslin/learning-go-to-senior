package retry_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	retry "github.com/holmeslin/learning-go-to-senior/examples/chapter5/08_synctest_or_clock/before"
)

// ANCHOR: example

func ExampleDo() {
	attempt := 0
	fn := func(context.Context) error {
		attempt++
		return fmt.Errorf("attempt %d: %w", attempt, errors.New("connection refused"))
	}
	// Backoff 為 0，所以不用等。
	err := retry.Do(context.Background(), retry.Policy{Attempts: 3, Timeout: time.Second}, fn)
	fmt.Println(err)
	// Output:
	// after 3 attempts: attempt 3: connection refused
}

// ANCHOR_END: example
