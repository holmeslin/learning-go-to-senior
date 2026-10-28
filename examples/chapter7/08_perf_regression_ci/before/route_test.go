package route_test

import (
	"testing"

	route "github.com/holmeslin/learning-go-to-senior/examples/chapter7/08_perf_regression_ci/before"
)

// ANCHOR: bench

func BenchmarkTemplate(b *testing.B) {
	p := "/users/48213/orders/90177/items/3"
	for b.Loop() {
		route.Template(p)
	}
}

// ANCHOR_END: bench
