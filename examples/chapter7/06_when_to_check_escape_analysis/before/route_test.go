package accesslog_test

import (
	"fmt"
	"testing"

	before "github.com/holmeslin/learning-go-to-senior/examples/chapter7/06_when_to_check_escape_analysis/before"
)

// ANCHOR: benchmark

func BenchmarkRoute(b *testing.B) {
	lines := []string{
		"GET /users/42 200",
		"POST /orders 201",
		"GET /healthz 503",
		"DELETE /users/42 404",
	}
	for b.Loop() {
		for _, line := range lines {
			before.Route(line)
		}
	}
}

// ANCHOR_END: benchmark

func ExampleRoute() {
	key, err := before.Route("GET /users/42 503")
	fmt.Println(key, err)

	_, err = before.Route("GET /users/42 99")
	fmt.Println(err)
	// Output:
	// GET /users/42 5xx <nil>
	// bad status "99"
}
