package track_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	track "github.com/holmeslin/learning-go-to-senior/examples/chapter9/05_context_propagation/after"
)

// provider 回傳一個把 span 收進記憶體的 TracerProvider。
func provider() (*sdktrace.TracerProvider, *tracetest.SpanRecorder) {
	rec := tracetest.NewSpanRecorder()
	return sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec)), rec
}

func noAudit(context.Context, string) {}

// ANCHOR: sametrace

func TestSpansShareOneTrace(t *testing.T) {
	t.Parallel()
	// 兩個服務各有自己的 provider 與 recorder，就像兩個行程
	carrierRec, gatewayRec := tracetest.NewSpanRecorder(), tracetest.NewSpanRecorder()
	carrierTP := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(carrierRec))
	gatewayTP := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(gatewayRec))
	logger := slog.New(slog.DiscardHandler)

	carrier := httptest.NewServer(track.NewCarrier(carrierTP, logger))
	defer carrier.Close()
	gw := track.NewGateway(gatewayTP, logger, carrier.Client(), carrier.URL, func(context.Context, string) {})
	front := httptest.NewServer(gw)
	defer front.Close()

	resp, err := front.Client().Get(front.URL + "/track?id=A1")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	gw.Wait()

	spans := map[string]sdktrace.ReadOnlySpan{}
	for _, s := range append(gatewayRec.Ended(), carrierRec.Ended()...) {
		spans[s.Name()] = s
	}
	if len(spans) != 4 {
		t.Fatalf("got %d spans, want 4", len(spans))
	}
	handler := spans["GET /track"]
	if handler.Parent().IsValid() {
		t.Error("GET /track should be the root span")
	}
	for child, parent := range map[string]string{
		"carrier.request": "GET /track",
		"carrier.lookup":  "carrier.request",
		"audit":           "GET /track",
	} {
		c, p := spans[child], spans[parent]
		if c.Parent().SpanID() != p.SpanContext().SpanID() {
			t.Errorf("%s: parent span is %s, want %s", child, c.Parent().SpanID(), parent)
		}
		if c.SpanContext().TraceID() != handler.SpanContext().TraceID() {
			t.Errorf("%s: trace ID differs from GET /track", child)
		}
	}
	if !spans["carrier.lookup"].Parent().IsRemote() {
		t.Error("carrier.lookup should have a remote parent")
	}
}

// ANCHOR_END: sametrace

func TestLogsCarryTraceID(t *testing.T) {
	t.Parallel()
	tp, rec := provider()
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	carrier := httptest.NewServer(track.NewCarrier(tp, logger))
	carrier.Close() // 承運商掛了
	gw := track.NewGateway(tp, logger, carrier.Client(), carrier.URL, noAudit)

	w := httptest.NewRecorder()
	gw.ServeHTTP(w, httptest.NewRequest("GET", "/track?id=A1", nil))
	gw.Wait()

	if w.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadGateway)
	}
	var line map[string]any
	if err := json.Unmarshal(buf.Bytes(), &line); err != nil {
		t.Fatalf("log is not one JSON line: %v\n%s", err, buf.String())
	}
	want := rec.Ended()[0].SpanContext().TraceID().String()
	if line["level"] != "ERROR" || line["trace_id"] != want {
		t.Errorf("log = %v, want level ERROR with trace_id %s", line, want)
	}
}

func TestBackgroundSurvivesRequestCancel(t *testing.T) {
	t.Parallel()
	tp, _ := provider()
	logger := slog.New(slog.DiscardHandler)
	carrier := httptest.NewServer(track.NewCarrier(tp, logger))
	defer carrier.Close()

	release, got := make(chan struct{}), make(chan error, 1)
	audit := func(ctx context.Context, _ string) {
		<-release // 等請求取消之後才檢查
		got <- ctx.Err()
	}
	gw := track.NewGateway(tp, logger, carrier.Client(), carrier.URL, audit)

	ctx, cancel := context.WithCancel(t.Context())
	req := httptest.NewRequestWithContext(ctx, "GET", "/track?id=A1", nil)
	gw.ServeHTTP(httptest.NewRecorder(), req)
	cancel() // 請求結束，context 取消
	close(release)
	gw.Wait()

	if err := <-got; err != nil {
		t.Errorf("background ctx.Err() = %v, want nil", err)
	}
}

// ANCHOR: link

func TestLinkedRoot(t *testing.T) {
	t.Parallel()
	rec := tracetest.NewSpanRecorder()
	tracer := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec)).Tracer("test")

	reqCtx, request := tracer.Start(t.Context(), "request")
	_, job := tracer.Start(context.WithoutCancel(reqCtx), "job",
		trace.WithNewRoot(), trace.WithLinks(trace.LinkFromContext(reqCtx)))
	job.End()
	request.End()

	ended := rec.Ended() // 先結束的在前：job、request
	job0, request0 := ended[0], ended[1]
	if job0.SpanContext().TraceID() == request0.SpanContext().TraceID() {
		t.Error("job should start a new trace")
	}
	if job0.Parent().IsValid() {
		t.Error("job should be a root span")
	}
	if links := job0.Links(); len(links) != 1 || links[0].SpanContext.SpanID() != request0.SpanContext().SpanID() {
		t.Errorf("links = %v, want one link to the request span", links)
	}
}

// ANCHOR_END: link
