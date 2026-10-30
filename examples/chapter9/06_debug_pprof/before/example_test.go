package stock_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	stock "github.com/holmeslin/learning-go-to-senior/examples/chapter9/06_debug_pprof/before"
)

// ANCHOR: example

// get 對 h 送出 GET 請求，回傳狀態碼與回應內容。
func get(h http.Handler, path string) (int, string) {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec.Code, rec.Body.String()
}

func ExampleNewHandler() {
	h := stock.NewHandler() // 對外的 handler：http.ListenAndServe(addr, nil) 用的就是它
	code, _ := get(h, "/stock/A-100")
	fmt.Println("GET /stock/A-100 =", code)
	code, body := get(h, "/debug/pprof/")
	fmt.Println("GET /debug/pprof/ =", code, "lists goroutine:", strings.Contains(body, "goroutine"))
	code, _ = get(h, "/debug/pprof/heap")
	fmt.Println("GET /debug/pprof/heap =", code)
	// Output:
	// GET /stock/A-100 = 200
	// GET /debug/pprof/ = 200 lists goroutine: true
	// GET /debug/pprof/heap = 200
}

// ANCHOR_END: example
