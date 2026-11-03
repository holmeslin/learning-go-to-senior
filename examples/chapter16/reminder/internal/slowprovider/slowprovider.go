// Package slowprovider 是壓測用的假推播供應商：每個請求先等一段時間，再依比例回 429，其餘回 200；
// 設了 Status，則每個請求都回那個狀態碼，模擬整個掛掉的供應商。
//
// 它模擬的是第 15 章事故裡壓測沒有模擬到的那種供應商：變慢、偶爾限流。只給 cmd/slowprovider 在本機使用，
// 服務的測試用的是假的 RoundTripper，不經過它。請求的內容不讀、不記錄：裝置 token 不該出現在壓測的日誌裡。
package slowprovider

import (
	"errors"
	"io"
	"math/rand/v2"
	"net/http"
	"time"
)

// Options 是假供應商的行為。
type Options struct {
	// Delay 是每個請求在回應之前等待的時間；0 代表不等。請求被取消（客戶端逾時、斷線）時立刻放棄，不再回應。
	Delay time.Duration
	// TooManyRequests 是回 429 的比例，0 到 1：0 全部回 200，1 全部回 429。
	TooManyRequests float64
	// Status 不為 0 時，每個請求等完 Delay 之後都回這個狀態碼（例如 503），不看 TooManyRequests 與 Rand；
	// 0 代表依 TooManyRequests 回 429 或 200。有值時必須是 100 到 599 的狀態碼。
	Status int
	// Rand 回傳 [0, 1) 的亂數，決定這個請求回 429 還是 200；nil 時用 math/rand/v2。
	// 測試注入固定的序列，才能斷言哪一個請求回了什麼。
	Rand func() float64
}

// NewHandler 回傳假供應商的 handler。只有 POST /push（推播客戶端呼叫的路徑），其他路徑與方法由 ServeMux 回 404 與 405。
// Options 不合法（Delay 為負、比例不在 0 到 1 之間、Status 不是 0 也不在 100 到 599 之間）時回傳錯誤。
func NewHandler(opts Options) (http.Handler, error) {
	if opts.Delay < 0 {
		return nil, errors.New("slowprovider: delay must not be negative")
	}
	if opts.TooManyRequests < 0 || opts.TooManyRequests > 1 {
		return nil, errors.New("slowprovider: the share of 429 responses must be between 0 and 1")
	}
	if opts.Status != 0 && (opts.Status < 100 || opts.Status > 599) {
		return nil, errors.New("slowprovider: status must be 0 or between 100 and 599")
	}
	rnd := opts.Rand
	if rnd == nil {
		rnd = rand.Float64
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /push", func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body) // 讀完本文，連線才能重用
		timer := time.NewTimer(opts.Delay)
		defer timer.Stop()
		select {
		case <-timer.C:
		case <-r.Context().Done():
			return
		}
		if opts.Status != 0 {
			w.WriteHeader(opts.Status)
			return
		}
		if rnd() < opts.TooManyRequests {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	return mux, nil
}
