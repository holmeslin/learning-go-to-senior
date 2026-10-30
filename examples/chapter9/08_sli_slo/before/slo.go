// Package slo 示範用 SDK 預設的直方圖 bucket 收延遲，再回頭算 SLI。
package slo

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

const durationName = "request.duration"

// ANCHOR: recorder

// Recorder 把每個請求的狀態碼類別與延遲（毫秒）記進直方圖。
type Recorder struct {
	hist metric.Float64Histogram
}

// NewRecorder 用 mp 建立延遲直方圖，沒有指定 bucket 邊界，用 SDK 的預設值。
func NewRecorder(mp metric.MeterProvider) (*Recorder, error) {
	h, err := mp.Meter("slo").Float64Histogram(durationName, metric.WithUnit("ms"))
	if err != nil {
		return nil, err
	}
	return &Recorder{hist: h}, nil
}

// Record 記下一個請求：狀態碼類別（例如 5xx）當屬性，延遲當量測值。
func (r *Recorder) Record(ctx context.Context, status int, d time.Duration) {
	class := attribute.String("status_class", fmt.Sprintf("%dxx", status/100))
	r.hist.Record(ctx, float64(d)/float64(time.Millisecond), metric.WithAttributes(class))
}

// ANCHOR_END: recorder

// ANCHOR: within

// Within 回傳在 threshold（毫秒）以內的請求數的下限與上限，以及請求總數。
// 門檻落在某個 bucket 裡面時，直方圖只知道那一格的請求數，不知道門檻的哪一邊，
// 所以只能給範圍：下限不含那一格，上限含整格。
func Within(rm metricdata.ResourceMetrics, threshold float64) (lo, hi, total int64) {
	for _, dp := range datapoints(rm) {
		total += int64(dp.Count)
		for i, n := range dp.BucketCounts {
			switch {
			case i < len(dp.Bounds) && dp.Bounds[i] <= threshold:
				lo += int64(n)
				hi += int64(n)
			case i == 0 || dp.Bounds[i-1] < threshold:
				hi += int64(n)
			}
		}
	}
	return lo, hi, total
}

// ANCHOR_END: within

// ANCHOR: alert

// Alert 在出現任何 5xx 時回報 true。
func Alert(rm metricdata.ResourceMetrics) bool {
	for _, dp := range datapoints(rm) {
		if v, _ := dp.Attributes.Value("status_class"); v.AsString() == "5xx" && dp.Count > 0 {
			return true
		}
	}
	return false
}

// ANCHOR_END: alert

func datapoints(rm metricdata.ResourceMetrics) []metricdata.HistogramDataPoint[float64] {
	var dps []metricdata.HistogramDataPoint[float64]
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if h, ok := m.Data.(metricdata.Histogram[float64]); ok && m.Name == durationName {
				dps = append(dps, h.DataPoints...)
			}
		}
	}
	return dps
}
