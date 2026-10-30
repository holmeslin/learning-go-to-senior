// Package loadshed 示範無上限地接受請求的 handler：超載時所有請求一起排隊。
package loadshed

import "net/http"

// ANCHOR: handler

// NewHandler 回傳一個照單全收的 handler。capacity 是真正做事的容量
// （CPU 核心數、資料庫連線池的大小），work 是每個請求要做的事；
// 超過 capacity 的請求都停在 slots 前排隊，沒有人被拒絕。
func NewHandler(capacity int, work func()) http.Handler {
	slots := make(chan struct{}, capacity)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slots <- struct{}{}
		defer func() { <-slots }()
		work()
		w.WriteHeader(http.StatusOK)
	})
}

// ANCHOR_END: handler
