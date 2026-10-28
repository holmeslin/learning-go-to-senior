package route_test

import (
	"fmt"

	route "github.com/holmeslin/learning-go-to-senior/examples/chapter7/08_perf_regression_ci/after"
)

func ExampleTemplate() {
	fmt.Println(route.Template("/users/48213/orders/90177/items/3"))
	fmt.Println(route.Template("/users/48213/profile"))
	fmt.Println(route.Template("/healthz"))
	// Output:
	// /users/:id/orders/:id/items/:id
	// /users/:id/profile
	// /healthz
}
