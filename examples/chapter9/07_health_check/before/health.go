// Package health 是常見寫法：單一個 /health，同時當 liveness 與 readiness。
package health

import (
	"fmt"
	"net/http"
)

// Pinger 是健康檢查要 ping 的資料庫；*sql.DB 就符合這個介面。
type Pinger interface {
	Ping() error
}

// ANCHOR: handler

// Handler 回傳 /health 的 handler：資料庫 ping 得通回 200，否則回 503。
func Handler(db Pinger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(); err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		fmt.Fprint(w, "ok")
	})
}

// ANCHOR_END: handler
