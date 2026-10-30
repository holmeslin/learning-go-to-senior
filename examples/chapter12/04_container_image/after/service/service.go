// Package service 是被打包進容器映像的示範服務：一個存活端點加一個打招呼的端點。
package service

import (
	"fmt"
	"net/http"
)

// NewHandler 回傳服務的路由：GET /livez 回 200 與 ok，GET /hello 回
// 「hello, 」加 name 查詢參數（沒給或是空字串時用 world）。
func NewHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /livez", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ok")
	})
	mux.HandleFunc("GET /hello", func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		if name == "" {
			name = "world"
		}
		fmt.Fprintf(w, "hello, %s\n", name)
	})
	return mux
}
