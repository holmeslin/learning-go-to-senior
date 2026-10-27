package store_test

import (
	"errors"
	"fmt"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter2/07_error_contract/after/store"
)

// ANCHOR: example

func Example() {
	_, err := store.Find(42)
	fmt.Println("not found:", errors.Is(err, store.ErrNotFound))

	_, err = store.Find(-1)
	if e, ok := errors.AsType[*store.InvalidIDError](err); ok {
		fmt.Println("invalid id:", e.ID)
	}
	// Output:
	// not found: true
	// invalid id: -1
}

// ANCHOR_END: example
