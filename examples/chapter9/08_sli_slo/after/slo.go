// Package slo 示範用包含 SLO 門檻的 bucket 邊界收延遲，並算出 SLI、錯誤預算與消耗速度。
package slo

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

const durationName = "http.server.request.duration"

const (
	// LatencyThreshold 是延遲 SLO 的門檻：在這個時間內完成的請求算「好」。
	LatencyThreshold = 300 * time.Millisecond
	// LatencyTarget 是延遲 SLO：99% 的請求要在 LatencyThreshold 內完成。
	LatencyTarget = 0.99
	// AvailabilityTarget 是可用性 SLO：99.9% 的請求不能是 5xx。
	AvailabilityTarget = 0.999
	// FastBurn 是告警的消耗速度。30 天的窗口裡以這個速度消耗 1 小時，
	// 用掉 2% 的預算（14.4 ÷ 720 小時）。
	FastBurn = 14.4
)

// ErrNoBoundary 表示直方圖的 bucket 邊界不含 LatencyThreshold，算不出確定的 SLI。
var ErrNoBoundary = errors.New("latency threshold is not a bucket boundary")

// ANCHOR: recorder

// Recorder 把每個請求的狀態碼類別與延遲（秒）記進直方圖。
type Recorder struct {
	hist metric.Float64Histogram
}

// NewRecorder 用 mp 建立延遲直方圖。bucket 邊界由 mp 的 View 決定，見 LatencyView。
func NewRecorder(mp metric.MeterProvider) (*Recorder, error) {
	h, err := mp.Meter("slo").Float64Histogram(durationName, metric.WithUnit("s"))
	if err != nil {
		return nil, err
	}
	return &Recorder{hist: h}, nil
}

// Record 記下一個請求：狀態碼類別（例如 5xx）當屬性，延遲當量測值。
func (r *Recorder) Record(ctx context.Context, status int, d time.Duration) {
	class := attribute.String("status_class", fmt.Sprintf("%dxx", status/100))
	r.hist.Record(ctx, d.Seconds(), metric.WithAttributes(class))
}

// LatencyView 讓延遲直方圖的 bucket 邊界包含 LatencyThreshold，
// 在建立 MeterProvider 時用 sdkmetric.WithView 傳入。
func LatencyView() sdkmetric.View {
	return sdkmetric.NewView(
		sdkmetric.Instrument{Name: durationName},
		sdkmetric.Stream{Aggregation: sdkmetric.AggregationExplicitBucketHistogram{
			Boundaries: []float64{0.05, 0.1, 0.2, LatencyThreshold.Seconds(), 0.5, 1, 2.5},
		}},
	)
}

// ANCHOR_END: recorder

// ANCHOR: result

// Result 是一個 SLO 在一批請求上的結果。
type Result struct {
	Good, Total int64
	// SLI 是好請求的比例；沒有請求時是 1。
	SLI float64
	// Allowed 是錯誤預算：這批請求最多允許幾個壞請求。
	Allowed float64
	// Remaining 是剩下的預算，負數代表已經超支。
	Remaining float64
	// BurnRate 是壞請求的比例除以允許的比例；1 代表剛好用完預算。
	BurnRate float64
}

func newResult(good, total int64, target float64) Result {
	r := Result{Good: good, Total: total, SLI: 1, Allowed: (1 - target) * float64(total)}
	if total == 0 {
		return r
	}
	bad := float64(total - good)
	r.SLI = float64(good) / float64(total)
	r.Remaining = r.Allowed - bad
	r.BurnRate = bad / r.Allowed
	return r
}

// Page 回報消耗速度是否達到 FastBurn，也就是該叫醒人的程度。
func (r Result) Page() bool { return r.BurnRate >= FastBurn }

// ANCHOR_END: result

// Report 是可用性與延遲兩個 SLO 的結果。
type Report struct {
	Availability, Latency Result
}

// ANCHOR: compute

// Compute 從 ManualReader 收到的資料算出兩個 SLI。
// 延遲直方圖的邊界不含 LatencyThreshold 時回傳 ErrNoBoundary，而不是估一個數字。
func Compute(rm metricdata.ResourceMetrics) (Report, error) {
	var total, notServerError, fast int64
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			h, ok := m.Data.(metricdata.Histogram[float64])
			if !ok || m.Name != durationName {
				continue
			}
			for _, dp := range h.DataPoints {
				i := slices.Index(dp.Bounds, LatencyThreshold.Seconds())
				if i < 0 {
					return Report{}, ErrNoBoundary
				}
				total += int64(dp.Count)
				if v, _ := dp.Attributes.Value("status_class"); v.AsString() != "5xx" {
					notServerError += int64(dp.Count)
				}
				for _, n := range dp.BucketCounts[:i+1] {
					fast += int64(n)
				}
			}
		}
	}
	return Report{
		Availability: newResult(notServerError, total, AvailabilityTarget),
		Latency:      newResult(fast, total, LatencyTarget),
	}, nil
}

// ANCHOR_END: compute
