// Package telemetry 組起提醒服務的指標與追蹤：一個 MeterProvider、一個 TracerProvider，
// 加上 goroutine 數量與 heap 兩個 runtime 指標。
//
// 指標、日誌與追蹤各回答不同的問題：指標回答「現在有多糟」（沒有高基數的標籤），
// 日誌回答「那一次發生了什麼」（不記裝置 token），追蹤回答「時間花在哪」。
// 這個套件只管指標與追蹤；兩者都不用全域設定（otel.SetMeterProvider），測試各自建自己的。
package telemetry

import (
	"context"
	"errors"
	"fmt"
	"io"
	"runtime/metrics"
	"slices"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// scope 是 meter 與 tracer 的名稱。
const scope = "reminder"

// Telemetry 是服務的指標與追蹤。
type Telemetry struct {
	// Meter 建立指標用。
	Meter metric.Meter
	// Tracer 建立 span 用。
	Tracer trace.Tracer

	reader *sdkmetric.ManualReader
	meters *sdkmetric.MeterProvider
	traces *sdktrace.TracerProvider
}

// ANCHOR: new

// New 建立 Telemetry，並註冊 goroutine 數量與 heap 兩個 runtime 指標。
//
// exporter 是追蹤送去的地方，Shutdown 時會把緩衝裡的 span 送完；nil 代表不送出。
//
// shortcut: 指標用 ManualReader，只有診斷端點被讀的時候才收集（Snapshot），代替真正的 exporter；
// 追蹤沒有預設的 exporter（OTLP 不在 go.mod 裡）。教材要的是 span 與指標的形狀，
// 上線時指標換成 PeriodicReader 加 exporter，追蹤在 main 接 OTLP exporter，Shutdown 的做法不變。
func New(exporter sdktrace.SpanExporter) (*Telemetry, error) {
	reader := sdkmetric.NewManualReader()
	meters := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	var traceOpts []sdktrace.TracerProviderOption
	if exporter != nil {
		traceOpts = append(traceOpts, sdktrace.WithBatcher(exporter))
	}
	traces := sdktrace.NewTracerProvider(traceOpts...)

	t := &Telemetry{
		Meter:  meters.Meter(scope),
		Tracer: traces.Tracer(scope),
		reader: reader, meters: meters, traces: traces,
	}
	err := errors.Join(
		runtimeGauge(t.Meter, "runtime.goroutines", "活著的 goroutine 數量", "{goroutine}", "/sched/goroutines:goroutines"),
		runtimeGauge(t.Meter, "runtime.heap.bytes", "heap 裡的物件佔用的記憶體", "By", "/memory/classes/heap/objects:bytes"),
	)
	if err != nil {
		return nil, errors.Join(err, t.Shutdown(context.Background()))
	}
	return t, nil
}

// ANCHOR_END: new

// runtimeGauge 註冊一個讀 runtime/metrics 的指標：每次收集時讀一次 sample。
func runtimeGauge(m metric.Meter, name, description, unit, sample string) error {
	_, err := m.Int64ObservableGauge(name,
		metric.WithDescription(description), metric.WithUnit(unit),
		metric.WithInt64Callback(func(_ context.Context, o metric.Int64Observer) error {
			s := []metrics.Sample{{Name: sample}}
			metrics.Read(s)
			if s[0].Value.Kind() == metrics.KindUint64 {
				o.Observe(int64(s[0].Value.Uint64()))
			}
			return nil
		}))
	if err != nil {
		return fmt.Errorf("register %s: %w", name, err)
	}
	return nil
}

// ANCHOR: shutdown

// Shutdown 送出最後的追蹤與指標，之後不能再用。關機時在關 HTTP 伺服器之後、關資料庫之前呼叫：
// 進行中的送出已經結束，它們的 span 與計數才是完整的；而收集 overdue 指標的查詢還需要資料庫。
// ctx 的期限就是送出的期限，期限到了回傳錯誤。
func (t *Telemetry) Shutdown(ctx context.Context) error {
	return errors.Join(t.traces.Shutdown(ctx), t.meters.Shutdown(ctx))
}

// ANCHOR_END: shutdown

// ANCHOR: snapshot

// Snapshot 收集一次指標，一個資料點一行（依名稱排序）寫進 w：
//
//	reminder.deliveries{result="sent"} 3
//	reminder.scan.duration count=1 sum=0.012
func (t *Telemetry) Snapshot(ctx context.Context, w io.Writer) error {
	var rm metricdata.ResourceMetrics
	if err := t.reader.Collect(ctx, &rm); err != nil {
		return fmt.Errorf("collect metrics: %w", err)
	}
	var lines []string
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			lines = append(lines, render(m)...)
		}
	}
	slices.Sort(lines)
	_, err := io.WriteString(w, strings.Join(lines, "\n")+"\n")
	return err
}

// render 把一個指標的每個資料點寫成一行；只認得本服務用到的三種。
func render(m metricdata.Metrics) []string {
	var lines []string
	switch d := m.Data.(type) {
	case metricdata.Sum[int64]:
		for _, dp := range d.DataPoints {
			lines = append(lines, fmt.Sprintf("%s%s %d", m.Name, labels(dp.Attributes), dp.Value))
		}
	case metricdata.Gauge[int64]:
		for _, dp := range d.DataPoints {
			lines = append(lines, fmt.Sprintf("%s%s %d", m.Name, labels(dp.Attributes), dp.Value))
		}
	case metricdata.Histogram[float64]:
		for _, dp := range d.DataPoints {
			lines = append(lines, fmt.Sprintf("%s%s count=%d sum=%g", m.Name, labels(dp.Attributes), dp.Count, dp.Sum))
		}
	default:
		lines = append(lines, fmt.Sprintf("# %s: unsupported data type %T", m.Name, m.Data))
	}
	return lines
}

// labels 回傳 {key="value",...}；沒有屬性時是空字串。
func labels(set attribute.Set) string {
	if set.Len() == 0 {
		return ""
	}
	var parts []string
	for iter := set.Iter(); iter.Next(); {
		kv := iter.Attribute()
		parts = append(parts, fmt.Sprintf("%s=%q", kv.Key, kv.Value.String()))
	}
	return "{" + strings.Join(parts, ",") + "}"
}

// ANCHOR_END: snapshot
