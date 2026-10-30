package stock_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	stock "github.com/holmeslin/learning-go-to-senior/examples/chapter9/06_debug_pprof/after"
)

// status 對 h 送出 GET 請求，回傳狀態碼。
func status(h http.Handler, path string) int {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec.Code
}

// ANCHOR: example

func ExampleNewAdminHandler() {
	public, admin := stock.NewPublicHandler(), stock.NewAdminHandler()
	fmt.Println("public /stock/A-100      =", status(public, "/stock/A-100"))
	fmt.Println("public /debug/pprof/     =", status(public, "/debug/pprof/"))
	fmt.Println("admin  /debug/pprof/     =", status(admin, "/debug/pprof/"))
	fmt.Println("admin  /debug/pprof/heap =", status(admin, "/debug/pprof/heap"))
	// Output:
	// public /stock/A-100      = 200
	// public /debug/pprof/     = 404
	// admin  /debug/pprof/     = 200
	// admin  /debug/pprof/heap = 200
}

// ANCHOR_END: example
