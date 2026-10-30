package slo_test

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	slo "github.com/holmeslin/learning-go-to-senior/examples/chapter9/08_sli_slo/before"
)

// ANCHOR: example

// collect 記 96 個 120 ms 的請求與 4 個花 tail 的請求，回傳 ManualReader 收到的資料。
func collect(tail time.Duration) metricdata.ResourceMetrics {
	ctx := context.Background()
	reader := metric.NewManualReader()
	rec, err := slo.NewRecorder(metric.NewMeterProvider(metric.WithReader(reader)))
	if err != nil {
		panic(err)
	}
	for range 96 {
		rec.Record(ctx, 200, 120*time.Millisecond)
	}
	for range 4 {
		rec.Record(ctx, 200, tail)
	}
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &rm); err != nil {
		panic(err)
	}
	return rm
}

func ExampleWithin() {
	for _, c := range []struct {
		name   string
		tail   time.Duration
		actual int
	}{
		{"A", 260 * time.Millisecond, 100},
		{"B", 400 * time.Millisecond, 96},
	} {
		lo, hi, total := slo.Within(collect(c.tail), 300)
		fmt.Printf("%s: histogram says %d..%d of %d within 300ms, actual %d\n", c.name, lo, hi, total, c.actual)
	}
	// Output:
	// A: histogram says 96..100 of 100 within 300ms, actual 100
	// B: histogram says 96..100 of 100 within 300ms, actual 96
}

// ANCHOR_END: example

// ANCHOR: alertexample

func ExampleAlert() {
	ctx := context.Background()
	reader := metric.NewManualReader()
	rec, err := slo.NewRecorder(metric.NewMeterProvider(metric.WithReader(reader)))
	if err != nil {
		panic(err)
	}
	for range 9999 {
		rec.Record(ctx, 200, 120*time.Millisecond)
	}
	rec.Record(ctx, 500, 120*time.Millisecond)
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &rm); err != nil {
		panic(err)
	}
	fmt.Println("1 error in 10000 requests, alert:", slo.Alert(rm))
	// Output: 1 error in 10000 requests, alert: true
}

// ANCHOR_END: alertexample
