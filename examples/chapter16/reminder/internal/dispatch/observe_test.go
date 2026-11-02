package dispatch_test

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter16/reminder/internal/push"
	"github.com/holmeslin/learning-go-to-senior/examples/chapter16/reminder/internal/store"
)

// mixed 讓活動 1 的 1 號永久失敗、2 號暫時失敗，其他成功。
func mixed(n push.Notification) error {
	switch n.Key {
	case "reminder-1-1":
		return fmt.Errorf("%w: provider returned 400", push.ErrPermanent)
	case "reminder-1-2":
		return fmt.Errorf("%w: provider returned 503", push.ErrTransient)
	}
	return nil
}

// ANCHOR: reader

// newReader 建一個只存在這個測試的 MeterProvider 與 ManualReader，不碰全域設定。
func newReader(t *testing.T) (*sdkmetric.ManualReader, *sdkmetric.MeterProvider) {
	t.Helper()
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() { mp.Shutdown(context.Background()) })
	return reader, mp
}

// collect 收一次指標，依名稱回傳。
func collect(t *testing.T, reader *sdkmetric.ManualReader) map[string]metricdata.Metrics {
	t.Helper()
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(t.Context(), &rm); err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	byName := map[string]metricdata.Metrics{}
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			byName[m.Name] = m
		}
	}
	return byName
}

// ANCHOR_END: reader

// ANCHOR: cardinality

// 送出結果的計數只有一個標籤 result，值是固定的七種：活動、報名者、裝置 token 都不是標籤，
// 不管這一批有多少人、多少場活動，資料點的數量都不會變。
func TestScanMetricsHaveOnlyTheResultLabel(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		st, db := setup(t, 5)
		addEvent(t, db, 2, time.Now().Add(store.LeadTime-90*time.Minute), 3) // 預定時間在 90 分鐘前：從未嘗試而逾期
		reader, mp := newReader(t)
		opts := options()
		opts.Meter = mp.Meter("test")
		d := newDispatcher(t, st, &fakeSender{behave: mixed}, opts)

		mustScan(t, d)
		metrics := collect(t, reader)

		sum, ok := metrics["reminder.deliveries"].Data.(metricdata.Sum[int64])
		if !ok {
			t.Fatalf("reminder.deliveries = %T, want a Sum[int64]", metrics["reminder.deliveries"].Data)
		}
		got := map[string]int64{}
		for _, dp := range sum.DataPoints {
			if dp.Attributes.Len() != 1 {
				t.Errorf("data point has attributes %v, want only result", dp.Attributes.ToSlice())
			}
			result, _ := dp.Attributes.Value("result")
			got[result.AsString()] = dp.Value
		}
		want := map[string]int64{"sent": 3, "retrying": 1, "failed": 1, "missed": 3, "expired": 0, "skipped": 0, "errored": 0}
		if len(got) != len(want) {
			t.Fatalf("results = %v, want exactly %v", got, want)
		}
		for result, n := range want {
			if got[result] != n {
				t.Errorf("deliveries{result=%q} = %d, want %d", result, got[result], n)
			}
		}

		// 每個指標、每個資料點：屬性的鍵只可能是 result。
		for name, m := range metrics {
			if name == "reminder.deliveries" {
				continue
			}
			if hist, ok := m.Data.(metricdata.Histogram[float64]); ok {
				for _, dp := range hist.DataPoints {
					if dp.Attributes.Len() != 0 {
						t.Errorf("%s has attributes %v, want none", name, dp.Attributes.ToSlice())
					}
				}
			}
		}
	})
}

// ANCHOR_END: cardinality

// 批次時間：每掃一批記一筆，單位是秒。
func TestScanRecordsTheBatchDuration(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		st, _ := setup(t, 4)
		reader, mp := newReader(t)
		opts := options()
		opts.Meter = mp.Meter("test")
		opts.Concurrency = 2
		d := newDispatcher(t, st, &fakeSender{delay: time.Second}, opts)

		mustScan(t, d) // 4 則、一次 2 則：兩輪，各一秒
		mustScan(t, d) // 沒有東西可送

		hist, ok := collect(t, reader)["reminder.scan.duration"].Data.(metricdata.Histogram[float64])
		if !ok || len(hist.DataPoints) != 1 {
			t.Fatalf("reminder.scan.duration = %+v, want one histogram data point", hist)
		}
		if dp := hist.DataPoints[0]; dp.Count != 2 || dp.Sum != 2 {
			t.Errorf("scan duration: count = %d, sum = %v, want 2 batches taking 2s in total", dp.Count, dp.Sum)
		}
	})
}

// ANCHOR: overdue

// 逾期未寄的數量在收集時才查：到期而沒有紀錄的、待重試的都算，寄出之後歸零。
func TestOverdueGaugeCountsWhatIsPastDueAndUnsent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		st, _ := setup(t, 3)
		reader, mp := newReader(t)
		var down atomic.Bool
		down.Store(true)
		sender := &fakeSender{behave: func(push.Notification) error {
			if down.Load() {
				return fmt.Errorf("%w: provider returned 503", push.ErrTransient)
			}
			return nil
		}}
		opts := options()
		opts.Meter = mp.Meter("test")
		opts.Jitter = func(limit time.Duration) time.Duration { return limit / 2 }
		d := newDispatcher(t, st, sender, opts)

		overdue := func() int64 {
			t.Helper()
			gauge, ok := collect(t, reader)["reminder.overdue.unsent"].Data.(metricdata.Gauge[int64])
			if !ok || len(gauge.DataPoints) != 1 {
				t.Fatalf("reminder.overdue.unsent = %+v, want one gauge data point", gauge)
			}
			return gauge.DataPoints[0].Value
		}
		if got := overdue(); got != 3 {
			t.Errorf("due but never attempted = %d, want 3", got)
		}
		mustScan(t, d)
		if got := overdue(); got != 3 {
			t.Errorf("after a scan where the provider is down = %d, want 3 (all retrying)", got)
		}
		down.Store(false)
		time.Sleep(10 * time.Minute) // 重試的時間到了
		mustScan(t, d)
		if got := overdue(); got != 0 {
			t.Errorf("after the retries were sent = %d, want 0", got)
		}
	})
}

// ANCHOR_END: overdue

func attrs(kvs []attribute.KeyValue) map[string]attribute.Value {
	m := map[string]attribute.Value{}
	for _, kv := range kvs {
		m[string(kv.Key)] = kv.Value
	}
	return m
}

// ANCHOR: spans

// 每批一個 span，每次送出一個子 span：子 span 屬於同一條追蹤、父親是這一批的 span。
// span 的屬性有活動編號、第幾次嘗試與結果；沒有裝置 token。
func TestScanRecordsOneSpanPerBatchWithOneChildPerSend(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		st, _ := setup(t, 4)
		exporter := tracetest.NewInMemoryExporter()
		tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
		opts := options()
		opts.Tracer = tp.Tracer("test")
		d := newDispatcher(t, st, &fakeSender{behave: mixed}, opts)

		mustScan(t, d)

		var scan tracetest.SpanStub
		var sends []tracetest.SpanStub
		for _, s := range exporter.GetSpans() {
			switch s.Name {
			case "reminder.scan":
				scan = s
			case "reminder.deliver":
				sends = append(sends, s)
			default:
				t.Errorf("unexpected span %q", s.Name)
			}
		}
		if scan.Name == "" || scan.Parent.IsValid() {
			t.Fatalf("scan span = %+v, want one root span named reminder.scan", scan)
		}
		if got := attrs(scan.Attributes); got["sent"].AsInt64() != 2 || got["retrying"].AsInt64() != 1 || got["failed"].AsInt64() != 1 {
			t.Errorf("scan span attributes = %v, want the batch summary", scan.Attributes)
		}
		if len(sends) != 4 {
			t.Fatalf("send spans = %d, want 4", len(sends))
		}
		var results []string
		for _, s := range sends {
			if s.Parent.SpanID() != scan.SpanContext.SpanID() || s.SpanContext.TraceID() != scan.SpanContext.TraceID() {
				t.Errorf("send span parent = %v, want the scan span %v", s.Parent.SpanID(), scan.SpanContext.SpanID())
			}
			a := attrs(s.Attributes)
			results = append(results, a["result"].AsString())
			if a["event.id"].AsInt64() != 1 || a["attempt"].AsInt64() != 1 {
				t.Errorf("send span attributes = %v, want event.id 1 and attempt 1", s.Attributes)
			}
			for _, kv := range s.Attributes {
				if strings.Contains(kv.Value.Emit(), "token-") { // setup 的裝置 token 都是 token-N
					t.Errorf("attribute %s = %q contains a device token", kv.Key, kv.Value.Emit())
				}
			}
			if failed := a["result"].AsString() == "failed"; failed != (s.Status.Code == codes.Error) {
				t.Errorf("result %q has status %v, want Error only for a permanent failure", a["result"].AsString(), s.Status.Code)
			}
		}
		slices.Sort(results)
		if want := []string{"failed", "retrying", "sent", "sent"}; !slices.Equal(results, want) {
			t.Errorf("send results = %v, want %v", results, want)
		}
	})
}

// ANCHOR_END: spans

// 沒有東西可送的批次也有一個 span，沒有子 span。
func TestScanWithNothingDueStillRecordsTheBatchSpan(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		st, _ := setup(t, 0)
		exporter := tracetest.NewInMemoryExporter()
		tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
		opts := options()
		opts.Tracer = tp.Tracer("test")
		d := newDispatcher(t, st, &fakeSender{}, opts)

		mustScan(t, d)
		if spans := exporter.GetSpans(); len(spans) != 1 || spans[0].Name != "reminder.scan" {
			t.Errorf("spans = %v, want just the scan span", spans)
		}
	})
}
