package accesslog_test

import (
	"fmt"

	accesslog "github.com/holmeslin/learning-go-to-senior/examples/chapter7/05_common_hotspots/after"
)

// ANCHOR: example

func ExampleReport() {
	fmt.Print(accesslog.Report([]accesslog.Entry{
		{Method: "GET", Path: "/Home?ref=ad", Status: 200},
		{Method: "POST", Path: "/API/Orders?id=7", Status: 500},
		{Method: "GET", Path: "/missing", Status: 404},
	}))
	// Output:
	// POST /api/orders 500
	// GET /missing 404
}

// ANCHOR_END: example
