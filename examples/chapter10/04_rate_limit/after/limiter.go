// Package ratelimit 用 golang.org/x/time/rate 限流：呼叫下游前等待，伺服器端依 client 分開、超過就回 429。
package ratelimit

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// ANCHOR: throttle

// Throttle 回傳限流後的 send：每次呼叫前先向 lim 取一個 token，
// 取不到就等；ctx 的 deadline 之前等不到，或 ctx 被取消，就不呼叫 send、直接回錯誤。
func Throttle(lim *rate.Limiter, send func(ctx context.Context, msg string) error) func(ctx context.Context, msg string) error {
	return func(ctx context.Context, msg string) error {
		if err := lim.Wait(ctx); err != nil {
			return fmt.Errorf("wait for rate limit: %w", err)
		}
		return send(ctx, msg)
	}
}

// ANCHOR_END: throttle

// ANCHOR: perclient

// PerClient 依 client 分開限流：每個 client 有自己的 token bucket。
//
// limiter 放在記憶體的 map 裡，client 越多 map 越大，而且沒有清理，只適合示範；
// 多個副本之間也不共用。零值不可用，請用 NewPerClient 建立。
type PerClient struct {
	limit rate.Limit
	burst int

	mu       sync.Mutex
	limiters map[string]*rate.Limiter
}

// NewPerClient 建立一個每個 client 每秒補 limit 個 token、最多累積 burst 個的限流器。
// limit 與 burst 都要大於 0。
func NewPerClient(limit rate.Limit, burst int) *PerClient {
	return &PerClient{limit: limit, burst: burst, limiters: map[string]*rate.Limiter{}}
}

func (p *PerClient) limiter(client string) *rate.Limiter {
	p.mu.Lock()
	defer p.mu.Unlock()
	lim, ok := p.limiters[client]
	if !ok {
		lim = rate.NewLimiter(p.limit, p.burst)
		p.limiters[client] = lim
	}
	return lim
}

// Handler 回傳包住 next 的 handler。client 以 X-Client-ID 標頭辨識（示範用，
// 沒帶標頭的請求共用一個 limiter）；超過限制就回 429，並用 Retry-After 告訴 client 幾秒後再來。
func (p *PerClient) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lim := p.limiter(r.Header.Get("X-Client-ID"))
		if !lim.Allow() {
			w.Header().Set("Retry-After", retryAfter(lim))
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// retryAfter 算出還要多久才有下一個 token，進位到整數秒（Retry-After 的最小單位）。
func retryAfter(lim *rate.Limiter) string {
	wait := time.Duration((1 - lim.Tokens()) / float64(lim.Limit()) * float64(time.Second))
	return strconv.Itoa(int(math.Ceil(wait.Seconds())))
}

// ANCHOR_END: perclient
