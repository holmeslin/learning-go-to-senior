// Package httpmetrics 是一個 HTTP 中介層，數每個請求並依屬性分組。
package httpmetrics

import (
	"net/http"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// ANCHOR: middleware

// Middleware 回傳會為每個請求計數的 handler。mp 由呼叫端注入。
// 屬性取自請求本身：方法、原始路徑、使用者代號。
func Middleware(mp metric.MeterProvider, next http.Handler) (http.Handler, error) {
	requests, err := mp.Meter("httpmetrics").Int64Counter("http.server.requests",
		metric.WithUnit("{request}"))
	if err != nil {
		return nil, err
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
		requests.Add(r.Context(), 1, metric.WithAttributes(
			attribute.String("http.request.method", r.Method),
			attribute.String("url.path", r.URL.Path),
			attribute.String("user.id", r.Header.Get("X-User-ID")),
		))
	}), nil
}

// ANCHOR_END: middleware
