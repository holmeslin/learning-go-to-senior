package batch_test

import (
	"context"
	"fmt"

	batch "github.com/holmeslin/learning-go-to-senior/examples/chapter6/06_worker_pool_or_goroutine_per_task/after"
)

// ANCHOR: example

func ExampleFetchAll() {
	fetch := func(ctx context.Context, id int) (string, error) {
		return fmt.Sprintf("page-%d", id), nil
	}
	titles, err := batch.FetchAll(context.Background(), []int{3, 1, 2}, 2, fetch)
	fmt.Println(titles, err)
	// Output:
	// [page-3 page-1 page-2] <nil>
}

// ANCHOR_END: example
