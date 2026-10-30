// Package stock 是一個庫存查詢服務：診斷端點用 import 順手掛上，服務也用 DefaultServeMux 對外。
package stock

import (
	"fmt"
	"net/http"
	_ "net/http/pprof" // 想在正式環境收 profile，順手 import 就有 /debug/pprof/
)

// ANCHOR: handler

func init() {
	http.HandleFunc("GET /stock/{sku}", lookup)
}

// NewHandler 回傳對外的 handler，也就是 http.DefaultServeMux。
// pprof 的 init 已經把診斷端點註冊在同一個 mux 上。
func NewHandler() http.Handler {
	return http.DefaultServeMux
}

// ANCHOR_END: handler

func lookup(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "%s: in stock\n", r.PathValue("sku"))
}
