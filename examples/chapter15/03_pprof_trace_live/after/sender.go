// Package reminder 是提醒 worker 的送出部分：把一批提醒送給推播供應商。
//
// 這是修正後的版本：每一次呼叫都有逾時，同時進行的請求有上限，失敗會回傳錯誤。
package reminder

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/sync/errgroup"
)

// 供應商一般 100 毫秒內回應、限制每秒 50 次：同時 8 個請求綽綽有餘；等 2 秒還沒回應就當它失敗。
// shortcut: 寫死成常數，正式環境要從設定讀（第 12 章〈設定與環境變數〉），範例不做。
const (
	maxInFlight    = 8
	requestTimeout = 2 * time.Second
)

// Push 是一則要送給推播供應商的提醒。
type Push struct {
	// Key 是「活動 + 報名者」組成的冪等鍵，供應商用它辨識重複的請求。
	Key string
}

// Sender 把提醒送給推播供應商。
type Sender struct {
	// Client 是呼叫供應商用的 HTTP client。
	Client *http.Client
	// URL 是供應商推播 API 的位址。
	URL string
}

// ANCHOR: sendall

// SendAll 送出全部提醒，等每一則都有結果才回傳。
// 同時進行的請求最多 maxInFlight 個，每個請求最多等 requestTimeout；
// 一則失敗不影響其他則，回傳的錯誤把所有失敗的提醒合在一起（errors.Join）。
func (s Sender) SendAll(ctx context.Context, pushes []Push) error {
	errs := make([]error, len(pushes)) // 每個 goroutine 只寫自己那一格，不需要鎖
	var g errgroup.Group
	g.SetLimit(maxInFlight)
	for i, p := range pushes {
		g.Go(func() error { // 滿了會在這裡等，不會一口氣開出全部 goroutine
			errs[i] = s.post(ctx, p)
			return nil // 錯誤放進 errs：回傳給 errgroup 會讓它只留第一個
		})
	}
	_ = g.Wait() // 上面的函式都回傳 nil；錯誤已經收進 errs
	return errors.Join(errs...)
}

// ANCHOR_END: sendall

// post 送出一則提醒。
func (s Sender) post(ctx context.Context, p Push) error {
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.URL, http.NoBody)
	if err != nil {
		return err
	}
	req.Header.Set("Idempotency-Key", p.Key)
	resp, err := s.Client.Do(req)
	if err != nil {
		return fmt.Errorf("push %s: %w", p.Key, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("push %s: supplier answered %s", p.Key, resp.Status)
	}
	return nil
}
