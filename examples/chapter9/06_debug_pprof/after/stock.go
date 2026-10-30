// Package stock 是一個庫存查詢服務：對外的 handler 與管理用的 handler 分開組裝。
package stock

import (
	"fmt"
	"net/http"
	"net/http/pprof"
)

// NewPublicHandler 回傳對外的 handler：只有業務路由，沒有任何診斷端點。
func NewPublicHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /stock/{sku}", lookup)
	return mux
}

// ANCHOR: admin

// NewAdminHandler 回傳管理用的 handler：只有明確註冊的 pprof 端點，沒有業務路由。
// 呼叫端只能把它交給只聽本機或內部網路的 listener。
func NewAdminHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /debug/pprof/", pprof.Index) // 也負責 heap、goroutine 等具名的 profile
	mux.HandleFunc("GET /debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("GET /debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("GET /debug/pprof/trace", pprof.Trace)
	return mux
}

// ANCHOR_END: admin

func lookup(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "%s: in stock\n", r.PathValue("sku"))
}
