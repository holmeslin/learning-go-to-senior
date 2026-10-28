// Package shophttp 把訂單服務透過 HTTP 提供出去。它只認得 order 套件，不碰 database/sql。
package shophttp

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter4/01_repository_layer/after/order"
)

// NewHandler 回傳訂單服務的 http.Handler。
func NewHandler(store *order.Store) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /orders/{id}", getOrder(store))
	mux.HandleFunc("GET /users/{id}/orders/open", listOpen(store))
	mux.HandleFunc("POST /users/{id}/close", closeAccount(store))
	return mux
}

// parseID 讀出路徑裡的 {id}；格式不對時直接回 400，並回傳 false。
func parseID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return 0, false
	}
	return id, true
}

// ANCHOR: get

func getOrder(store *order.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := parseID(w, r)
		if !ok {
			return
		}
		o, err := store.Find(r.Context(), id)
		if errors.Is(err, order.ErrNotFound) {
			http.Error(w, "order not found", http.StatusNotFound)
			return
		}
		if err != nil {
			log.Printf("get order %d: %v", id, err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "%d user=%d %s %s\n", o.ID, o.UserID, o.Item, o.Status)
	}
}

// ANCHOR_END: get

func listOpen(store *order.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := parseID(w, r)
		if !ok {
			return
		}
		orders, err := store.OpenOrdersFor(r.Context(), userID)
		if err != nil {
			log.Printf("list open orders of user %d: %v", userID, err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		for _, o := range orders {
			fmt.Fprintf(w, "%d %s %s\n", o.ID, o.Item, o.Status)
		}
	}
}

// ANCHOR: close

func closeAccount(store *order.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := parseID(w, r)
		if !ok {
			return
		}
		open, err := store.OpenOrdersFor(r.Context(), userID)
		if err != nil {
			log.Printf("list open orders of user %d: %v", userID, err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if len(open) > 0 {
			http.Error(w, "user has open orders", http.StatusConflict)
			return
		}
		// 真正關閉帳號的部分省略。
		w.WriteHeader(http.StatusNoContent)
	}
}

// ANCHOR_END: close
