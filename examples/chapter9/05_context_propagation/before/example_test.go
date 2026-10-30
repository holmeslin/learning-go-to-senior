package track_test

import (
	"context"
	"fmt"
	"net/http/httptest"
	"slices"
	"strings"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	track "github.com/holmeslin/learning-go-to-senior/examples/chapter9/05_context_propagation/before"
)

// ANCHOR: example

func ExampleGateway() {
	rec := tracetest.NewSpanRecorder() // 兩個服務共用一個 recorder，一次看完所有 span
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec))

	carrier := httptest.NewServer(track.NewCarrier(tp))
	defer carrier.Close()
	gw := track.NewGateway(tp, carrier.Client(), carrier.URL, func(context.Context, string) {})
	gw.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/track?id=A1", nil))
	gw.Wait()

	spans := rec.Ended()
	names := map[trace.SpanID]string{}
	for _, s := range spans {
		names[s.SpanContext().SpanID()] = s.Name()
	}
	slices.SortFunc(spans, func(a, b sdktrace.ReadOnlySpan) int { return strings.Compare(a.Name(), b.Name()) })
	traces := map[trace.TraceID]int{} // trace ID 依序編號：1、2、3……
	for _, s := range spans {
		id := s.SpanContext().TraceID()
		if traces[id] == 0 {
			traces[id] = len(traces) + 1
		}
		parent := "(root)"
		if s.Parent().IsValid() {
			parent = names[s.Parent().SpanID()]
		}
		fmt.Printf("%-16s parent=%-16s trace=%d\n", s.Name(), parent, traces[id])
	}
	// Output:
	// GET /track       parent=(root)           trace=1
	// audit            parent=(root)           trace=2
	// carrier.lookup   parent=(root)           trace=3
	// carrier.request  parent=GET /track       trace=1
}

// ANCHOR_END: example
