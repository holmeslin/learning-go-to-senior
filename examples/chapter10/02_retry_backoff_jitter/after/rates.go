// Package rates 向匯率服務查詢：暫時性的失敗用有上限的指數退避加 jitter 重試，
// 其他失敗立刻回報。
package rates

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"strconv"
	"syscall"
	"time"
)

// StatusError 表示下游回了非 200 的狀態碼。
type StatusError struct {
	Code       int           // HTTP 狀態碼
	RetryAfter time.Duration // 下游用 Retry-After 要求的等待；沒有標頭時是 0
}

func (e *StatusError) Error() string { return "unexpected status " + strconv.Itoa(e.Code) }

// ANCHOR: policy

// Policy 是重試的規則。零值只嘗試一次，不設逾時，也不等待。
type Policy struct {
	MaxAttempts int           // 含第一次在內最多嘗試幾次；小於 1 時當成 1
	Timeout     time.Duration // 每次嘗試的逾時；0 代表不另設
	Base        time.Duration // 第一次重試前，等待時間的上限；之後每次加倍
	Max         time.Duration // 等待時間上限的天花板（不含 Retry-After）；0 代表不設天花板

	// Jitter 回傳 [0, limit) 之間的等待時間。nil 時用 math/rand/v2 的全域亂數。
	// 這個欄位是注入亂數來源的接縫，讓測試能用固定 seed 的 *rand.Rand 或固定的值；
	// 注入的函式要自己負責並行安全。
	Jitter func(limit time.Duration) time.Duration
}

// wait 是第 attempt 次失敗（從 0 起算）之後要等多久：full jitter 的隨機值，
// 加上下游要求的 retryAfter。
func (p Policy) wait(attempt int, retryAfter time.Duration) time.Duration {
	limit := p.Base << min(attempt, 30)
	if p.Max > 0 && limit > p.Max {
		limit = p.Max
	}
	var jitter time.Duration
	switch {
	case limit <= 0:
	case p.Jitter != nil:
		jitter = p.Jitter(limit)
	default:
		jitter = rand.N(limit)
	}
	return retryAfter + jitter
}

// ANCHOR_END: policy

// Client 向匯率服務發 GET，並依 Policy 重試。可以在多個 goroutine 之間共用。
type Client struct {
	HTTP   *http.Client // nil 時用 http.DefaultClient
	Policy Policy
}

// ANCHOR: fetch

// Fetch 取得 url 的內容。逾時、連線被拒、429 與 503 才會重試；其他錯誤（例如 400）
// 立刻回傳。ctx 結束，或下一次嘗試前的等待會超過 ctx 的 deadline 時，也立刻回傳。
func (c *Client) Fetch(ctx context.Context, url string) ([]byte, error) {
	for attempt := 0; ; attempt++ {
		body, err := c.once(ctx, url)
		if err == nil {
			return body, nil
		}
		if !retryable(err) {
			return nil, err
		}
		if attempt+1 >= max(c.Policy.MaxAttempts, 1) {
			return nil, fmt.Errorf("giving up after %d attempts: %w", attempt+1, err)
		}

		var retryAfter time.Duration
		if se, ok := errors.AsType[*StatusError](err); ok {
			retryAfter = se.RetryAfter
		}
		wait := c.Policy.wait(attempt, retryAfter)
		if dl, ok := ctx.Deadline(); ok && time.Until(dl) <= wait {
			return nil, fmt.Errorf("next attempt would pass the deadline: %w", err)
		}
		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return nil, fmt.Errorf("%w; last error: %w", ctx.Err(), err)
		}
	}
}

// retryable 判斷錯誤是不是暫時性的：單次嘗試逾時、連線被拒、429 與 503。
func retryable(err error) bool {
	if se, ok := errors.AsType[*StatusError](err); ok {
		return se.Code == http.StatusTooManyRequests || se.Code == http.StatusServiceUnavailable
	}
	return errors.Is(err, context.DeadlineExceeded) || errors.Is(err, syscall.ECONNREFUSED)
}

// ANCHOR_END: fetch

// once 發一次請求，套用 Policy.Timeout。Retry-After 只讀秒數的寫法，日期的寫法當成沒有。
func (c *Client) once(ctx context.Context, url string) ([]byte, error) {
	if c.Policy.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.Policy.Timeout)
		defer cancel()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	hc := c.HTTP
	if hc == nil {
		hc = http.DefaultClient
	}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		secs, _ := strconv.Atoi(resp.Header.Get("Retry-After"))
		return nil, &StatusError{Code: resp.StatusCode, RetryAfter: time.Duration(max(secs, 0)) * time.Second}
	}
	return io.ReadAll(resp.Body)
}
