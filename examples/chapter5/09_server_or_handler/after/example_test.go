package notes_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	notes "github.com/holmeslin/learning-go-to-senior/examples/chapter5/09_server_or_handler/after"
)

// do 對 h 送出一個請求，回傳狀態碼。auth 不是空字串時放進 Authorization 標頭。
func do(h http.Handler, method, path, auth string) int {
	req := httptest.NewRequest(method, path, nil)
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code
}

// ANCHOR: example

func ExampleNewHandler() {
	h := notes.NewHandler("s3cret", map[string]string{"1": "buy milk"})
	fmt.Println("GET /notes/1 =", do(h, "GET", "/notes/1", ""))
	fmt.Println("DELETE /notes/1 without token =", do(h, "DELETE", "/notes/1", ""))
	// Output:
	// GET /notes/1 = 200
	// DELETE /notes/1 without token = 401
}

// ANCHOR_END: example
