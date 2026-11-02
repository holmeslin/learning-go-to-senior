// Package push 是推播供應商的客戶端：送出一則通知，把失敗分成永久與暫時兩類。
//
// 分類是這個套件和呼叫端之間的契約：呼叫端只用 errors.Is 判斷 ErrPermanent 或 ErrTransient，
// 決定放棄還是重試，不看狀態碼。
package push

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ANCHOR: errors

var (
	// ErrPermanent 表示重試也不會成功：供應商拒絕了這則通知（4xx，429 除外），
	// 或請求本身有問題。呼叫端應該放棄。
	ErrPermanent = errors.New("push: permanent failure")
	// ErrTransient 表示稍後可能成功：429、5xx、逾時與其他連線錯誤。呼叫端可以退避後重試。
	ErrTransient = errors.New("push: transient failure")
)

// ANCHOR_END: errors

// Notification 是要送給一台裝置的通知。
type Notification struct {
	Key         string // 冪等鍵，帶在 Idempotency-Key 標頭；同一則通知的每次重試都用同一個
	DeviceToken string // 目標裝置，不可寫進日誌
	EventID     int64  // 活動編號，App 用它開啟對應的頁面
	Title       string // 通知標題
	Body        string // 通知內文
}

type payload struct {
	DeviceToken string `json:"device_token"`
	EventID     int64  `json:"event_id"`
	Title       string `json:"title"`
	Body        string `json:"body"`
}

// Client 呼叫推播供應商。可以在多個 goroutine 之間共用。
type Client struct {
	endpoint string
	timeout  time.Duration
	http     *http.Client
}

// NewClient 回傳呼叫 baseURL 的客戶端。timeout 是每個請求的逾時；
// rt 是底層的傳輸，nil 時用 http.DefaultTransport，測試用假的 RoundTripper 取代網路。
//
// 逾時用每個請求的 context，不設 http.Client.Timeout：期限只有一個來源，錯誤也帶著 context.DeadlineExceeded。
func NewClient(baseURL string, timeout time.Duration, rt http.RoundTripper) *Client {
	return &Client{
		endpoint: strings.TrimRight(baseURL, "/") + "/push",
		timeout:  timeout,
		http: &http.Client{
			Transport: rt,
			// 不跟著轉址：請求的本文有裝置 token，不送到沒設定過的地方。3xx 會被當成永久失敗。
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}
}

// ANCHOR: send

// Send 送出 n，成功回傳 nil。失敗的錯誤一定包著 ErrPermanent 或 ErrTransient 其中之一：
//
//   - 2xx：成功。
//   - 429 與 5xx：暫時失敗，供應商忙或限流，等一下再試可能就好。
//   - 其他 4xx 與 3xx：永久失敗，重送同樣的內容不會有不同的結果。
//   - 逾時與其他連線錯誤：暫時失敗。請求可能已經到了供應商，所以重試要用同一個冪等鍵。
//   - 建不出請求（網址有問題）：永久失敗。
func (c *Client) Send(ctx context.Context, n Notification) error {
	body, err := json.Marshal(payload{DeviceToken: n.DeviceToken, EventID: n.EventID, Title: n.Title, Body: n.Body})
	if err != nil {
		return fmt.Errorf("encode notification: %w: %w", ErrPermanent, err)
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w: %w", ErrPermanent, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", n.Key)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("send notification: %w: %w", ErrTransient, err)
	}
	defer resp.Body.Close()
	// 讀掉（有上限）回應本文，連線才能被重用；內容不使用。
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4<<10))

	switch code := resp.StatusCode; {
	case code >= 200 && code < 300:
		return nil
	case code == http.StatusTooManyRequests || code >= 500:
		return fmt.Errorf("provider returned %d: %w", code, ErrTransient)
	default:
		return fmt.Errorf("provider returned %d: %w", code, ErrPermanent)
	}
}

// ANCHOR_END: send
