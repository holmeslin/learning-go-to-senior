// Package httpmetrics 是一個 HTTP 中介層，用有限的屬性集合記錄請求的次數與耗時，
// 並把連線池的統計匯出成非同步的量測值。
package httpmetrics

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// ANCHOR: middleware

// Middleware 回傳會記錄請求次數與耗時的 handler。mp 由呼叫端注入。
// 屬性只有三個，值都來自有限的集合：方法（不認得的歸成 _OTHER）、
// 路由樣板（對不上路由的歸成 unmatched）、狀態碼類別（2xx、4xx 等）。
// 它必須包在 mux 外面：mux 比對完路由之後才會設定 r.Pattern。
func Middleware(mp metric.MeterProvider, next http.Handler) (http.Handler, error) {
	meter := mp.Meter("httpmetrics")
	requests, err := meter.Int64Counter("http.server.requests",
		metric.WithUnit("{request}"))
	if err != nil {
		return nil, err
	}
	duration, err := meter.Float64Histogram("http.server.request.duration",
		metric.WithUnit("s"),
		// SDK 預設的邊界是為毫秒設計的（0 到 10000）；耗時用秒記，要自己給邊界。
		metric.WithExplicitBucketBoundaries(0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10))
	if err != nil {
		return nil, err
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		attrs := metric.WithAttributes(
			attribute.String("http.request.method", method(r.Method)),
			attribute.String("http.route", route(r.Pattern)),
			attribute.String("http.response.status_class", strconv.Itoa(rec.status/100)+"xx"),
		)
		requests.Add(r.Context(), 1, attrs)
		duration.Record(r.Context(), time.Since(start).Seconds(), attrs)
	}), nil
}

// ANCHOR_END: middleware

// ANCHOR: bounds

// method 回傳標準方法本身，其他的（客戶端想送什麼都行）一律是 _OTHER。
func method(m string) string {
	switch m {
	case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut,
		http.MethodPatch, http.MethodDelete, http.MethodOptions, http.MethodConnect, http.MethodTrace:
		return m
	}
	return "_OTHER"
}

// route 把 mux 的樣板（例如 "GET /users/{id}"）轉成路由；沒有比對到的回傳 unmatched。
func route(pattern string) string {
	if pattern == "" {
		return "unmatched"
	}
	if _, path, ok := strings.Cut(pattern, " "); ok {
		return path
	}
	return pattern
}

// ANCHOR_END: bounds

// statusRecorder 記下 handler 寫出的狀態碼。
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// Unwrap 讓 http.ResponseController 找得到底下的 ResponseWriter。
func (r *statusRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }
