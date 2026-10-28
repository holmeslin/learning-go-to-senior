package hits_test

import (
	"fmt"
	"sync"

	hits "github.com/holmeslin/learning-go-to-senior/examples/chapter6/07_channel_or_mutex/after"
)

func ExampleCounter() {
	var c hits.Counter // 零值就能用，不需要建構函式，也不需要 Close

	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			for range 250 {
				c.Inc("/home")
			}
			c.Inc("/about")
		})
	}
	wg.Wait()

	fmt.Println("/home", c.Count("/home"))
	fmt.Println("/about", c.Count("/about"))
	fmt.Println("/missing", c.Count("/missing"))
	// Output:
	// /home 1000
	// /about 4
	// /missing 0
}
