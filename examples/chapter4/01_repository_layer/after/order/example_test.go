package order_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter4/01_repository_layer/after/order"
)

// ANCHOR: notfound

func ExampleStore_Find_notFound() {
	db := mustOpenDB("ExampleStore_Find_notFound")
	defer db.Close()
	store := order.NewStore(db)

	_, err := store.Find(context.Background(), 99)
	fmt.Println(err)
	fmt.Println(errors.Is(err, order.ErrNotFound))
	fmt.Println(errors.Is(err, sql.ErrNoRows))
	// Output:
	// find order 99: order not found
	// true
	// false
}

// ANCHOR_END: notfound

// ANCHOR: open

func ExampleStore_OpenOrdersFor() {
	db := mustOpenDB("ExampleStore_OpenOrdersFor")
	defer db.Close()
	seed(db, 1, "keyboard", "paid")
	seed(db, 1, "mouse", "pending")
	seed(db, 1, "cable", "shipped")
	seed(db, 2, "monitor", "pending")
	store := order.NewStore(db)

	orders, err := store.OpenOrdersFor(context.Background(), 1)
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, o := range orders {
		fmt.Println(o.ID, o.Item, o.Status)
	}
	// Output:
	// 1 keyboard paid
	// 2 mouse pending
}

// ANCHOR_END: open
