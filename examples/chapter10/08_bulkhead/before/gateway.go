// Package bulkhead 讓商品頁的後端呼叫多個下游。這個版本的所有下游共用同一組 worker 與同一個 HTTP client。
package bulkhead

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

// ANCHOR: gateway

// Gateway 用同一組 worker 與同一個 client 呼叫所有下游。
type Gateway struct {
	slots  chan struct{} // 容量 = 所有下游合計能同時進行的請求數
	client *http.Client
	urls   map[string]string
}

// New 建立 Gateway。workers 是所有下游合計的同時請求上限，至少 1；urls 把下游名稱對應到位址。
func New(workers int, client *http.Client, urls map[string]string) (*Gateway, error) {
	if workers < 1 {
		return nil, fmt.Errorf("workers must be at least 1, got %d", workers)
	}
	return &Gateway{slots: make(chan struct{}, workers), client: client, urls: urls}, nil
}

// Get 向名為 name 的下游發 GET，回傳回應內容。拿不到 worker 時等到 ctx 結束。
func (g *Gateway) Get(ctx context.Context, name, path string) (string, error) {
	base, ok := g.urls[name]
	if !ok {
		return "", fmt.Errorf("unknown downstream %q", name)
	}
	select {
	case g.slots <- struct{}{}:
		defer func() { <-g.slots }()
	case <-ctx.Done():
		return "", ctx.Err()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+path, nil)
	if err != nil {
		return "", err
	}
	resp, err := g.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("downstream %q returned %s", name, resp.Status)
	}
	return string(body), nil
}

// ANCHOR_END: gateway
