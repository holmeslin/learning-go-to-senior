package health_test

import (
	"fmt"
	"time"

	health "github.com/holmeslin/learning-go-to-senior/examples/chapter9/07_health_check/after"
)

// ANCHOR: example

func Example() {
	h := health.NewHandler(
		health.Check{Name: "db", Timeout: time.Millisecond, Probe: hangs},
		health.Check{Name: "cache", Timeout: time.Second, Probe: refuses},
	)

	for _, path := range []string{"/livez", "/readyz"} {
		code, body := get(h, path)
		fmt.Println(path, code, body)
	}
	// Output:
	// /livez 200 ok
	// /readyz 503 {"status":"unavailable","failed":{"cache":"connection refused","db":"timeout"}}
}

// ANCHOR_END: example
