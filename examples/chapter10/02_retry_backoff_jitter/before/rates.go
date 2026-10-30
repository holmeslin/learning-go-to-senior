// Package rates 向匯率服務查詢：失敗就再試，直到成功。
package rates

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ANCHOR: fetch

// Fetch 取得 url 的內容；失敗就再試，直到成功或 ctx 結束。
// base 大於 0 時，第 n 次失敗之後等 base<<n；base 是 0 就立刻重試。
func Fetch(ctx context.Context, hc *http.Client, url string, base time.Duration) ([]byte, error) {
	for attempt := 0; ctx.Err() == nil; attempt++ {
		body, err := get(ctx, hc, url)
		if err == nil {
			return body, nil
		}
		if base > 0 {
			select {
			case <-time.After(base << attempt):
			case <-ctx.Done():
			}
		}
	}
	return nil, ctx.Err()
}

// ANCHOR_END: fetch

func get(ctx context.Context, hc *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}
