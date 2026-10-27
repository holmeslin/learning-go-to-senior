package task_test

import (
	"errors"
	"fmt"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter1/01_package_by_responsibility/after/task"
)

// ANCHOR: example

func ExampleStore_Complete() {
	store := task.NewStore()
	t := store.Add("寫第 1 章")

	if err := store.Complete(t.ID); err != nil {
		fmt.Println("unexpected:", err)
		return
	}
	got, _ := store.Get(t.ID)
	fmt.Println(got.Title, got.Done)

	err := store.Complete(99)
	fmt.Println(errors.Is(err, task.ErrNotFound))
	// Output:
	// 寫第 1 章 true
	// true
}

// ANCHOR_END: example
