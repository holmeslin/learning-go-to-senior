// Package bulkhead 讓商品頁的後端呼叫多個下游。每個下游有自己的並行上限與 HTTP client，一個下游卡住不會占用其他下游的名額。
package bulkhead

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// ErrInvalidLimit 表示下游的 Limit 小於 1。
var ErrInvalidLimit = errors.New("limit must be at least 1")

// ErrUnknownDownstream 表示呼叫了沒有登記過的下游名稱。
var ErrUnknownDownstream = errors.New("unknown downstream")

// ANCHOR: gateway

// Downstream 描述一個下游：位址、專屬的 client、同時最多幾個請求。
type Downstream struct {
	BaseURL string
	Client  *http.Client // 每個下游各自一個，連線池不共用
	Limit   int          // 這個下游同時進行的請求上限，至少 1
}

// Gateway 依下游名稱，把請求送進各自的並行上限與 client。
type Gateway struct {
	lanes map[string]lane
}

// lane 是一個下游專屬的資源：設定加上它自己的名額。
type lane struct {
	Downstream
	slots chan struct{}
}

// New 建立 Gateway，每個下游得到容量為 Limit 的專屬名額。
func New(downstreams map[string]Downstream) (*Gateway, error) {
	g := &Gateway{lanes: make(map[string]lane, len(downstreams))}
	for name, d := range downstreams {
		if d.Limit < 1 {
			return nil, fmt.Errorf("downstream %q: %w, got %d", name, ErrInvalidLimit, d.Limit)
		}
		g.lanes[name] = lane{Downstream: d, slots: make(chan struct{}, d.Limit)}
	}
	return g, nil
}

// Get 向名為 name 的下游發 GET，回傳回應內容。這個下游的名額滿了就等，等到 ctx 結束。
func (g *Gateway) Get(ctx context.Context, name, path string) (string, error) {
	l, ok := g.lanes[name]
	if !ok {
		return "", fmt.Errorf("%w: %q", ErrUnknownDownstream, name)
	}
	select {
	case l.slots <- struct{}{}:
		defer func() { <-l.slots }()
	case <-ctx.Done():
		return "", ctx.Err()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, l.BaseURL+path, nil)
	if err != nil {
		return "", err
	}
	resp, err := l.Client.Do(req)
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

// ANCHOR: client

// NewClient 回傳一個專屬於單一下游的 client。它有自己的 Transport，閒置連線的上限和
// 這個下游的 Limit 一樣；預設只留 2 條，Limit 較大時，每次尖峰過後都要重新建立連線。
func NewClient(limit int) *http.Client {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxIdleConnsPerHost = limit
	return &http.Client{Transport: t}
}

// ANCHOR_END: client
