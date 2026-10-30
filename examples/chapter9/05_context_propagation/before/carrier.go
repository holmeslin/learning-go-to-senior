package track

import (
	"fmt"
	"net/http"

	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// NewCarrier 回傳承運商服務的 handler：從請求標頭取出上游的 trace，再開自己的 span。
func NewCarrier(tp trace.TracerProvider) http.Handler {
	tracer := tp.Tracer("carrier")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := propagation.TraceContext{}.Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		_, span := tracer.Start(ctx, "carrier.lookup")
		defer span.End()

		fmt.Fprintln(w, "in transit")
	})
}
