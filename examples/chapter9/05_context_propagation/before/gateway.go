// Package track 是出貨追蹤服務：gateway 收到查詢後向承運商（carrier）服務
// 查狀態，回覆之後在背景留一筆稽核紀錄。before 版的 trace 在兩個地方斷掉。
package track

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"

	"go.opentelemetry.io/otel/trace"
)

// Gateway 是對外的追蹤服務，實作 http.Handler。
type Gateway struct {
	tracer     trace.Tracer
	client     *http.Client
	carrierURL string
	audit      func(ctx context.Context, id string)
	bg         sync.WaitGroup
}

// NewGateway 建立 Gateway。TracerProvider 由呼叫端注入；audit 是回覆之後在背景執行的稽核。
func NewGateway(tp trace.TracerProvider, client *http.Client, carrierURL string, audit func(ctx context.Context, id string)) *Gateway {
	return &Gateway{tp.Tracer("track"), client, carrierURL, audit, sync.WaitGroup{}}
}

// Wait 等所有背景工作結束。
func (g *Gateway) Wait() { g.bg.Wait() }

// ANCHOR: handler

// ServeHTTP 處理 GET /track?id=...：向承運商查詢，回覆後在背景稽核。
func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := g.tracer.Start(r.Context(), "GET /track")
	defer span.End()
	id := r.URL.Query().Get("id")

	status, err := g.lookup(ctx, id)
	if err != nil {
		http.Error(w, "bad gateway", http.StatusBadGateway)
		return
	}

	g.bg.Go(func() {
		ctx, span := g.tracer.Start(context.Background(), "audit")
		defer span.End()
		g.audit(ctx, id)
	})
	fmt.Fprintln(w, status)
}

// lookup 向承運商查狀態。ctx 有傳進請求，但沒有把 trace 寫進標頭。
func (g *Gateway) lookup(ctx context.Context, id string) (string, error) {
	ctx, span := g.tracer.Start(ctx, "carrier.request")
	defer span.End()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.carrierURL+"?id="+url.QueryEscape(id), nil)
	if err != nil {
		return "", err
	}
	resp, err := g.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	return string(body), err
}

// ANCHOR_END: handler
