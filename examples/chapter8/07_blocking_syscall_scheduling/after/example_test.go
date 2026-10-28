package sourceread_test

import (
	"context"
	"fmt"

	sourceread "github.com/holmeslin/learning-go-to-senior/examples/chapter8/07_blocking_syscall_scheduling/after"
)

// ANCHOR: example

func ExampleReadAll() {
	sources := make([]int, 20)
	var c counter
	entered := make(chan struct{}, len(sources))
	release := make(chan struct{})
	read := func(ctx context.Context, n int) (int, error) {
		c.enter()
		defer c.leave()
		entered <- struct{}{}
		<-release // 模擬卡在 syscall：放行之前不會回來
		return 1, nil
	}

	done := make(chan int)
	go func() {
		total, _ := sourceread.ReadAll(context.Background(), sources, 4, read)
		done <- total
	}()

	for range 4 {
		<-entered // 上限 4：先等前 4 個讀取進來，才放行
	}
	close(release)
	fmt.Println("sources read:", <-done)
	fmt.Println("peak concurrent reads:", c.Peak())
	// Output:
	// sources read: 20
	// peak concurrent reads: 4
}

// ANCHOR_END: example
