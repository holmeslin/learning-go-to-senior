// Package track 是出貨追蹤服務：gateway 收到查詢後向承運商（carrier）服務
// 查狀態，回覆之後在背景留一筆稽核紀錄。after 版把 trace 一路傳下去。
package track

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sync"

	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// Gateway 是對外的追蹤服務，實作 http.Handler。
type Gateway struct {
	tracer     trace.Tracer
	log        *slog.Logger
	client     *http.Client
	carrierURL string
	audit      func(ctx context.Context, id string)
	bg         sync.WaitGroup
}

// NewGateway 建立 Gateway。TracerProvider 由呼叫端注入；audit 是回覆之後在背景執行的稽核。
func NewGateway(tp trace.TracerProvider, log *slog.Logger, client *http.Client, carrierURL string, audit func(ctx context.Context, id string)) *Gateway {
	return &Gateway{tp.Tracer("track"), log, client, carrierURL, audit, sync.WaitGroup{}}
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
		WithTrace(ctx, g.log).Error("carrier lookup failed", slog.String(KeyTrackingID, id), slog.Any(KeyError, err))
		http.Error(w, "bad gateway", http.StatusBadGateway)
		return
	}
	WithTrace(ctx, g.log).Info("tracked", slog.String(KeyTrackingID, id))

	// 背景工作要在請求結束後繼續：不帶走取消，但保留 trace。
	bgCtx := context.WithoutCancel(ctx)
	g.bg.Go(func() {
		bgCtx, span := g.tracer.Start(bgCtx, "audit")
		defer span.End()
		g.audit(bgCtx, id)
	})
	fmt.Fprintln(w, status)
}

// lookup 向承運商查狀態，並在對外的請求上注入 traceparent。
func (g *Gateway) lookup(ctx context.Context, id string) (string, error) {
	ctx, span := g.tracer.Start(ctx, "carrier.request")
	defer span.End()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.carrierURL+"?id="+url.QueryEscape(id), nil)
	if err != nil {
		return "", err
	}
	propagation.TraceContext{}.Inject(ctx, propagation.HeaderCarrier(req.Header))
	resp, err := g.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	return string(body), err
}

// ANCHOR_END: handler

// ANCHOR: withtrace

// 日誌鍵名，和本章〈日誌層級與欄位設計〉的約定一致：snake_case，錯誤用 error。
const (
	// KeyTrackingID 是貨件的追蹤編號。
	KeyTrackingID = "tracking_id"
	// KeyTraceID 是目前 span 所屬的 trace ID。
	KeyTraceID = "trace_id"
	// KeyError 是錯誤。
	KeyError = "error"
)

// WithTrace 回傳帶著 ctx 目前 span 的 trace_id 的 logger；ctx 裡沒有 span 時原樣回傳 l。
func WithTrace(ctx context.Context, l *slog.Logger) *slog.Logger {
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		return l.With(slog.String(KeyTraceID, sc.TraceID().String()))
	}
	return l
}

// ANCHOR_END: withtrace
