package slo_test

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	slo "github.com/holmeslin/learning-go-to-senior/examples/chapter9/08_sli_slo/after"
)

// ANCHOR: traffic

// traffic 是固定的請求資料：n 個回 status、花 latency 的請求。
type traffic struct {
	status  int
	latency time.Duration
	n       int
}

// collect 把 reqs 記進新的 MeterProvider，回傳 ManualReader 收到的資料。
func collect(reqs []traffic, opts ...sdkmetric.Option) (metricdata.ResourceMetrics, error) {
	ctx := context.Background()
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(append(opts, sdkmetric.WithReader(reader))...)
	rec, err := slo.NewRecorder(mp)
	if err != nil {
		return metricdata.ResourceMetrics{}, err
	}
	for _, r := range reqs {
		for range r.n {
			rec.Record(ctx, r.status, r.latency)
		}
	}
	var rm metricdata.ResourceMetrics
	err = reader.Collect(ctx, &rm)
	return rm, err
}

// ANCHOR_END: traffic

// ANCHOR: table

// near 比較兩個浮點數，容許極小的誤差。
func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestCompute(t *testing.T) {
	ms := time.Millisecond
	tests := []struct {
		name string
		reqs []traffic
		// 可用性與延遲各自的 SLI、剩餘預算與消耗速度。
		avail, latency         float64
		availLeft, latencyLeft float64
		availBurn, latencyBurn float64
	}{
		{"quiet", []traffic{{200, 120 * ms, 9999}, {500, 120 * ms, 1}}, 0.9999, 1, 9, 100, 0.1, 0},
		{"error burst", []traffic{{200, 120 * ms, 9800}, {503, 50 * ms, 200}}, 0.98, 1, -190, 100, 20, 0},
		{"slow tail", []traffic{{200, 120 * ms, 9850}, {200, 400 * ms, 150}}, 1, 0.985, 10, -50, 0, 1.5},
		{"exactly at threshold", []traffic{{200, 300 * ms, 100}}, 1, 1, 0.1, 1, 0, 0},
		{"no traffic", nil, 1, 1, 0, 0, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rm, err := collect(tt.reqs, sdkmetric.WithView(slo.LatencyView()))
			if err != nil {
				t.Fatal(err)
			}
			got, err := slo.Compute(rm)
			if err != nil {
				t.Fatal(err)
			}
			for _, c := range []struct {
				kind            string
				r               slo.Result
				sli, left, burn float64
			}{
				{"availability", got.Availability, tt.avail, tt.availLeft, tt.availBurn},
				{"latency", got.Latency, tt.latency, tt.latencyLeft, tt.latencyBurn},
			} {
				if !near(c.r.SLI, c.sli) || !near(c.r.Remaining, c.left) || !near(c.r.BurnRate, c.burn) {
					t.Errorf("%s = sli %v remaining %v burn %v, want %v %v %v",
						c.kind, c.r.SLI, c.r.Remaining, c.r.BurnRate, c.sli, c.left, c.burn)
				}
			}
		})
	}
}

// ANCHOR_END: table

func TestShouldPage(t *testing.T) {
	t.Parallel()
	ms := time.Millisecond
	for _, tt := range []struct {
		name string
		reqs []traffic
		want bool
	}{
		{"one error in 10000", []traffic{{200, 120 * ms, 9999}, {500, 120 * ms, 1}}, false},
		{"error burst", []traffic{{200, 120 * ms, 9800}, {503, 50 * ms, 200}}, true},
		{"slow tail burns 1.5x", []traffic{{200, 120 * ms, 9850}, {200, 400 * ms, 150}}, false},
	} {
		rm, err := collect(tt.reqs, sdkmetric.WithView(slo.LatencyView()))
		if err != nil {
			t.Fatal(err)
		}
		got, err := slo.Compute(rm)
		if err != nil {
			t.Fatal(err)
		}
		if page := got.Availability.ShouldPage() || got.Latency.ShouldPage(); page != tt.want {
			t.Errorf("%s: page = %t, want %t", tt.name, page, tt.want)
		}
	}
}

// ANCHOR: noboundary

func TestComputeNeedsBoundary(t *testing.T) {
	t.Parallel()
	reqs := []traffic{{200, 120 * time.Millisecond, 10}}
	rm, err := collect(reqs) // 沒有 View：SDK 預設的邊界裡沒有 0.3 秒
	if err != nil {
		t.Fatal(err)
	}
	if _, err := slo.Compute(rm); !errors.Is(err, slo.ErrNoBoundary) {
		t.Fatalf("Compute without view: err = %v, want ErrNoBoundary", err)
	}
}

// ANCHOR_END: noboundary
