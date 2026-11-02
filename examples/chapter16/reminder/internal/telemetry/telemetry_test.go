package telemetry_test

import (
	"bytes"
	"context"
	"slices"
	"strings"
	"testing"
	"testing/synctest"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter16/reminder/internal/telemetry"
)

func snapshot(t *testing.T, tel *telemetry.Telemetry) []string {
	t.Helper()
	var buf bytes.Buffer
	if err := tel.Snapshot(t.Context(), &buf); err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}
	return strings.Split(strings.TrimSpace(buf.String()), "\n")
}

// 快照一個資料點一行，依名稱排序，輸出是確定的。
func TestSnapshotRendersEachInstrumentKind(t *testing.T) {
	t.Parallel()
	tel, err := telemetry.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	counter, _ := tel.Meter.Int64Counter("test.counter")
	counter.Add(ctx, 3, metric.WithAttributes(attribute.String("result", "sent")))
	counter.Add(ctx, 1, metric.WithAttributes(attribute.String("result", "failed")))
	hist, _ := tel.Meter.Float64Histogram("test.histogram")
	hist.Record(ctx, 0.25)
	hist.Record(ctx, 0.5)
	gauge, _ := tel.Meter.Int64Gauge("test.gauge")
	gauge.Record(ctx, 7)

	var got []string
	for _, line := range snapshot(t, tel) {
		if strings.HasPrefix(line, "test.") {
			got = append(got, line)
		}
	}
	want := []string{
		`test.counter{result="failed"} 1`,
		`test.counter{result="sent"} 3`,
		`test.gauge 7`,
		`test.histogram count=2 sum=0.75`,
	}
	if !slices.Equal(got, want) {
		t.Errorf("snapshot lines = %q, want %q", got, want)
	}
}

// goroutine 數量與 heap 來自 runtime/metrics；快照裡有這兩項，值是數字（不斷言大小）。
func TestSnapshotIncludesRuntimeGauges(t *testing.T) {
	t.Parallel()
	tel, err := telemetry.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	lines := snapshot(t, tel)
	for _, name := range []string{"runtime.goroutines ", "runtime.heap.bytes "} {
		if !slices.ContainsFunc(lines, func(l string) bool { return strings.HasPrefix(l, name) }) {
			t.Errorf("snapshot has no %q line: %q", name, lines)
		}
	}
}

// ANCHOR: flush

// keepSpans 是 Shutdown 之後仍留著 span 的記憶體 exporter：tracetest.InMemoryExporter 的 Shutdown 會清空已收到的 span，
// 而要檢查的正是 Shutdown 送出了什麼。
type keepSpans struct{ *tracetest.InMemoryExporter }

func (keepSpans) Shutdown(context.Context) error { return nil }

// 關機時送出最後一段追蹤：span 先在批次處理器的緩衝裡，要等到批次的時間到或 Shutdown 才會送到 exporter。
// bubble 裡的時間是假的，「還沒送出」與「Shutdown 之後送出了」都是確定的。
func TestShutdownFlushesBufferedSpans(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		exporter := keepSpans{tracetest.NewInMemoryExporter()}
		tel, err := telemetry.New(exporter)
		if err != nil {
			t.Fatal(err)
		}
		_, span := tel.Tracer.Start(t.Context(), "last scan")
		span.End()
		synctest.Wait()
		if got := len(exporter.GetSpans()); got != 0 {
			t.Fatalf("spans exported before Shutdown = %d, want 0 (still buffered)", got)
		}

		if err := tel.Shutdown(t.Context()); err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
		if got := exporter.GetSpans(); len(got) != 1 || got[0].Name != "last scan" {
			t.Errorf("spans exported after Shutdown = %v, want the buffered span", got)
		}
	})
}

// ANCHOR_END: flush
