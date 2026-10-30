package track_test

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	track "github.com/holmeslin/learning-go-to-senior/examples/chapter9/05_context_propagation/after"
)

// ANCHOR: ids

// fixedIDs 產生固定的 trace ID 與依序遞增的 span ID，讓輸出每次都一樣。
type fixedIDs struct{ spans atomic.Uint64 }

func (g *fixedIDs) NewIDs(ctx context.Context) (trace.TraceID, trace.SpanID) {
	return trace.TraceID{0: 1, 1: 2, 2: 3, 3: 4, 4: 5, 5: 6, 6: 7, 7: 8, 8: 9, 9: 10, 10: 11, 11: 12, 12: 13, 13: 14, 14: 15, 15: 16}, g.NewSpanID(ctx, trace.TraceID{})
}

func (g *fixedIDs) NewSpanID(context.Context, trace.TraceID) trace.SpanID {
	return trace.SpanID{7: byte(g.spans.Add(1))}
}

// ANCHOR_END: ids

// ANCHOR: example

func ExampleGateway() {
	tp := sdktrace.NewTracerProvider(sdktrace.WithIDGenerator(&fixedIDs{}))
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{} // 拿掉時間，輸出才固定
			}
			return a
		},
	}))

	carrier := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("traceparent:", r.Header.Get("traceparent"))
		track.NewCarrier(tp, logger).ServeHTTP(w, r)
	}))
	defer carrier.Close()

	audit := func(ctx context.Context, id string) {
		track.WithTrace(ctx, logger).Info("audit", "id", id)
	}
	gw := track.NewGateway(tp, logger, carrier.Client(), carrier.URL, audit)
	gw.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/track?id=A1", nil))
	gw.Wait()
	// Output:
	// traceparent: 00-0102030405060708090a0b0c0d0e0f10-0000000000000002-01
	// level=INFO msg="carrier lookup" trace_id=0102030405060708090a0b0c0d0e0f10 id=A1
	// level=INFO msg=tracked trace_id=0102030405060708090a0b0c0d0e0f10 id=A1
	// level=INFO msg=audit trace_id=0102030405060708090a0b0c0d0e0f10 id=A1
}

// ANCHOR_END: example
