package shophttp_test

import (
	"net/http"
	"testing"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter4/01_repository_layer/after/order"
	"github.com/holmeslin/learning-go-to-senior/examples/chapter4/01_repository_layer/after/shophttp"
)

func TestGetOrder(t *testing.T) {
	db := newTestDB(t)
	seed(db, 7, "keyboard", "paid")
	h := shophttp.NewHandler(order.NewStore(db))

	tests := []struct {
		name string
		path string
		want string
	}{
		{"存在的訂單", "/orders/1", "200 1 user=7 keyboard paid"},
		{"不存在的訂單", "/orders/99", "404 order not found"},
		{"id 不是數字", "/orders/abc", "400 bad id"},
	}
	for _, tt := range tests {
		if got := call(h, http.MethodGet, tt.path); got != tt.want {
			t.Errorf("%s: GET %s = %q; want %q", tt.name, tt.path, got, tt.want)
		}
	}
}

func TestCloseAccount(t *testing.T) {
	db := newTestDB(t)
	seed(db, 1, "keyboard", "paid")
	seed(db, 2, "mouse", "pending")
	seed(db, 3, "cable", "shipped")
	h := shophttp.NewHandler(order.NewStore(db))

	tests := []struct {
		name string
		path string
		want string
	}{
		{"只有已付款的訂單", "/users/1/close", "409 user has open orders"},
		{"只有待處理的訂單", "/users/2/close", "409 user has open orders"},
		{"訂單都已出貨", "/users/3/close", "204"},
		{"沒有訂單", "/users/4/close", "204"},
	}
	for _, tt := range tests {
		if got := call(h, http.MethodPost, tt.path); got != tt.want {
			t.Errorf("%s: POST %s = %q; want %q", tt.name, tt.path, got, tt.want)
		}
	}
}

func TestDatabaseFailureIsNotNotFound(t *testing.T) {
	db := newTestDB(t)
	h := shophttp.NewHandler(order.NewStore(db))
	if err := db.Close(); err != nil { // 資料庫關了，之後的查詢都會失敗
		t.Fatalf("close db: %v", err)
	}

	for _, path := range []string{"/orders/1", "/users/1/orders/open"} {
		if got, want := call(h, http.MethodGet, path), "500 internal error"; got != want {
			t.Errorf("GET %s = %q; want %q", path, got, want)
		}
	}
	if got, want := call(h, http.MethodPost, "/users/1/close"), "500 internal error"; got != want {
		t.Errorf("POST /users/1/close = %q; want %q", got, want)
	}
}
