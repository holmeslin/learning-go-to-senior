package track

import (
	"fmt"
	"log/slog"
	"net/http"

	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// ANCHOR: carrier

// NewCarrier 回傳承運商服務的 handler：先從請求標頭取出上游的 trace，
// 再開自己的 span，所以它的 span 接在 gateway 的 span 底下。
func NewCarrier(tp trace.TracerProvider, log *slog.Logger) http.Handler {
	tracer := tp.Tracer("carrier")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := propagation.TraceContext{}.Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		ctx, span := tracer.Start(ctx, "carrier.lookup")
		defer span.End()

		WithTrace(ctx, log).Info("carrier lookup", "id", r.URL.Query().Get("id"))
		fmt.Fprintln(w, "in transit")
	})
}

// ANCHOR_END: carrier
