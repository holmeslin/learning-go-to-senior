package order_test

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"testing"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter4/01_repository_layer/after/order"
)

// ANCHOR: test

func TestFindTranslatesNoRows(t *testing.T) {
	db := newTestDB(t)

	_, err := order.NewStore(db).Find(context.Background(), 99)

	if !errors.Is(err, order.ErrNotFound) {
		t.Errorf("errors.Is(err, ErrNotFound) = false; err = %v", err)
	}
	if errors.Is(err, sql.ErrNoRows) {
		t.Error("errors.Is(err, sql.ErrNoRows) = true; the cause should not leak")
	}
}

// ANCHOR_END: test

func TestFindReturnsOrder(t *testing.T) {
	db := newTestDB(t)
	seed(db, 7, "keyboard", "paid")

	got, err := order.NewStore(db).Find(context.Background(), 1)
	if err != nil {
		t.Fatalf("Find() error = %v", err)
	}
	if want := (order.Order{ID: 1, UserID: 7, Item: "keyboard", Status: "paid"}); got != want {
		t.Errorf("Find() = %+v; want %+v", got, want)
	}
}

func TestOpenOrdersForSkipsShippedAndOtherUsers(t *testing.T) {
	db := newTestDB(t)
	seed(db, 1, "keyboard", "paid")
	seed(db, 1, "mouse", "pending")
	seed(db, 1, "cable", "shipped")
	seed(db, 2, "monitor", "pending")

	got, err := order.NewStore(db).OpenOrdersFor(context.Background(), 1)
	if err != nil {
		t.Fatalf("OpenOrdersFor() error = %v", err)
	}
	var items []string
	for _, o := range got {
		items = append(items, o.Item)
	}
	if want := []string{"keyboard", "mouse"}; !slices.Equal(items, want) {
		t.Errorf("items = %v; want %v", items, want)
	}
}

func TestOpenOrdersForNoOrders(t *testing.T) {
	db := newTestDB(t)

	got, err := order.NewStore(db).OpenOrdersFor(context.Background(), 1)

	if err != nil {
		t.Errorf("OpenOrdersFor() error = %v; want nil, an empty list is not an error", err)
	}
	if len(got) != 0 {
		t.Errorf("OpenOrdersFor() = %v; want empty", got)
	}
}

func TestOtherFailuresAreNotNotFound(t *testing.T) {
	db := newTestDB(t)
	store := order.NewStore(db)
	if err := db.Close(); err != nil { // 資料庫關了，之後的查詢都會失敗
		t.Fatalf("close db: %v", err)
	}

	_, findErr := store.Find(context.Background(), 1)
	_, openErr := store.OpenOrdersFor(context.Background(), 1)

	for name, err := range map[string]error{"Find": findErr, "OpenOrdersFor": openErr} {
		if err == nil {
			t.Errorf("%s() error = nil; want an error from the closed database", name)
		}
		if errors.Is(err, order.ErrNotFound) {
			t.Errorf("%s() error = %v; a database failure is not \"not found\"", name, err)
		}
	}
}

func TestCanceledContextIsKept(t *testing.T) {
	db := newTestDB(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := order.NewStore(db).Find(ctx, 1)

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Find() error = %v; want it to wrap context.Canceled", err)
	}
	if errors.Is(err, order.ErrNotFound) {
		t.Errorf("Find() error = %v; a canceled query is not \"not found\"", err)
	}
}
