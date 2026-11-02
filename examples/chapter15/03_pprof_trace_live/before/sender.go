// Package reminder 是提醒 worker 的送出部分：把一批提醒送給推播供應商。
//
// 這是事故版本，問題是刻意保留的，不要照抄：每則提醒開一個 goroutine、沒有並行上限；
// 請求沒有逾時，worker 傳進來的 ctx 也沒有 deadline；SendAll 不等結果，失敗也沒有人收。
package reminder

import (
	"context"
	"fmt"
	"net/http"
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

// SendAll 把每則提醒交給一個新的 goroutine 送出，然後立刻回傳 nil。
func (s Sender) SendAll(ctx context.Context, pushes []Push) error {
	for _, p := range pushes {
		go s.post(ctx, p) // post 回傳的錯誤沒有人收
	}
	return nil
}

// ANCHOR_END: sendall

// post 送出一則提醒。
func (s Sender) post(ctx context.Context, p Push) error {
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
