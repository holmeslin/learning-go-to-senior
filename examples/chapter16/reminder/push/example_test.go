package push_test

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter16/reminder/push"
)

// ANCHOR: example

func Example() {
	for _, code := range []int{http.StatusBadRequest, http.StatusTooManyRequests, http.StatusServiceUnavailable} {
		err := send(status(code))
		fmt.Printf("%d: %v\n", code, err)
		fmt.Printf("  permanent: %v, transient: %v\n", errors.Is(err, push.ErrPermanent), errors.Is(err, push.ErrTransient))
	}
	// Output:
	// 400: provider returned 400: push: permanent failure
	//   permanent: true, transient: false
	// 429: provider returned 429: push: transient failure
	//   permanent: false, transient: true
	// 503: provider returned 503: push: transient failure
	//   permanent: false, transient: true
}

// ANCHOR_END: example
