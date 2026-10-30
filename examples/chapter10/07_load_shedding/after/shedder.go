// Package loadshed 示範依同時處理數提早拒絕請求，並讓非核心功能降級。
package loadshed

import (
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// ANCHOR: shedder

// Priority 是請求的重要性，決定超載時先拒絕誰。
type Priority int

const (
	// Critical 是核心請求（例如結帳報價），最後才會被拒絕。
	Critical Priority = iota
	// Optional 是可以先犧牲的請求（例如推薦），超載時先被拒絕或降級。
	Optional
)

// Shedder 依同時處理數決定接受或拒絕請求，狀態只在這個值裡，不共用套件變數。
// 請用 NewShedder 建立。
type Shedder struct {
	limit, reserve int
	retryAfter     string

	mu       sync.Mutex
	inflight int
}

// NewShedder 建立 Shedder。limit 是同時處理數的上限；其中 reserve 個名額
// 只留給 Critical，Optional 最多用 limit-reserve 個（reserve 應小於 limit）。
// 被拒絕的回應用 retryAfter（無條件進位到整數秒）當 Retry-After。
func NewShedder(limit, reserve int, retryAfter time.Duration) *Shedder {
	return &Shedder{
		limit:      limit,
		reserve:    reserve,
		retryAfter: strconv.Itoa(int(math.Ceil(retryAfter.Seconds()))),
	}
}

// acquire 在有名額時佔用一個，否則回傳 false。
func (s *Shedder) acquire(p Priority) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	ceiling := s.limit
	if p == Optional {
		ceiling -= s.reserve
	}
	if s.inflight >= ceiling {
		return false
	}
	s.inflight++
	return true
}

func (s *Shedder) release() {
	s.mu.Lock()
	s.inflight--
	s.mu.Unlock()
}

// ANCHOR_END: shedder

// ANCHOR: handler

// Handler 包住 next：有名額就交給 next，並在它結束時歸還；沒有名額就不排隊，
// 立刻回應。fallback 為 nil 時回 503 與 Retry-After；
// 不為 nil 時改由 fallback 回應（應該是便宜的快取或部分結果），
// 並加上 X-Degraded: overload，讓呼叫端知道這不是完整的回應。
// fallback 不佔名額。
func (s *Shedder) Handler(p Priority, next, fallback http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.acquire(p) {
			defer s.release()
			next.ServeHTTP(w, r)
			return
		}
		if fallback != nil {
			w.Header().Set("X-Degraded", "overload")
			fallback.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Retry-After", s.retryAfter)
		http.Error(w, "overloaded", http.StatusServiceUnavailable)
	})
}

// ANCHOR_END: handler
