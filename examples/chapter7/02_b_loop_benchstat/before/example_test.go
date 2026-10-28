package ipfilter_test

import (
	"fmt"

	ipfilter "github.com/holmeslin/learning-go-to-senior/examples/chapter7/02_b_loop_benchstat/before"
)

// ANCHOR: example

func ExampleFilter() {
	ips := []string{"10.0.0.1", "203.0.113.7", "10.0.0.2", "198.51.100.9"}
	blocklist := []string{"203.0.113.7", "198.51.100.9"}
	fmt.Println(ipfilter.Filter(ips, blocklist))
	fmt.Println(ipfilter.IsInternal("10.1.2.3"), ipfilter.IsInternal("203.0.113.7"))
	// Output:
	// [10.0.0.1 10.0.0.2]
	// true false
}

// ANCHOR_END: example
