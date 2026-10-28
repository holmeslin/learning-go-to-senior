// Package shop 是常見寫法：每個 HTTP handler 自己寫 SQL，業務規則也直接夾在查詢旁邊。
package shop

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
)

// NewHandler 回傳訂單服務的 http.Handler。呼叫端要先開好 db 並建好 orders 資料表。
func NewHandler(db *sql.DB) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /orders/{id}", getOrder(db))
	mux.HandleFunc("GET /users/{id}/orders/open", listOpen(db))
	mux.HandleFunc("POST /users/{id}/close", closeAccount(db))
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

func getOrder(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := parseID(w, r)
		if !ok {
			return
		}
		var userID int64
		var item, status string
		err := db.QueryRowContext(r.Context(),
			`SELECT user_id, item, status FROM orders WHERE id = ?`, id,
		).Scan(&userID, &item, &status)
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "order not found", http.StatusNotFound)
			return
		}
		if err != nil {
			log.Printf("get order %d: %v", id, err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "%d user=%d %s %s\n", id, userID, item, status)
	}
}

// ANCHOR_END: get

// ANCHOR: list

func listOpen(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := parseID(w, r)
		if !ok {
			return
		}
		rows, err := db.QueryContext(r.Context(),
			`SELECT id, item, status FROM orders
			 WHERE user_id = ? AND status IN ('pending', 'paid') ORDER BY id`, userID)
		if err != nil {
			log.Printf("list open orders of user %d: %v", userID, err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		var body strings.Builder
		for rows.Next() {
			var id int64
			var item, status string
			if err := rows.Scan(&id, &item, &status); err != nil {
				log.Printf("scan open order of user %d: %v", userID, err)
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			fmt.Fprintf(&body, "%d %s %s\n", id, item, status)
		}
		if err := rows.Err(); err != nil {
			log.Printf("list open orders of user %d: %v", userID, err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		fmt.Fprint(w, body.String())
	}
}

// ANCHOR_END: list

// ANCHOR: close

func closeAccount(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := parseID(w, r)
		if !ok {
			return
		}
		var open int
		err := db.QueryRowContext(r.Context(),
			`SELECT COUNT(*) FROM orders WHERE user_id = ? AND status = 'pending'`, userID,
		).Scan(&open)
		if err != nil {
			log.Printf("count open orders of user %d: %v", userID, err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if open > 0 {
			http.Error(w, "user has open orders", http.StatusConflict)
			return
		}
		// 真正關閉帳號的部分省略。
		w.WriteHeader(http.StatusNoContent)
	}
}

// ANCHOR_END: close
