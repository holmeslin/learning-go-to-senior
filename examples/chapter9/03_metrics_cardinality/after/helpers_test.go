package httpmetrics_test

import (
	"context"
	"slices"

	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// ANCHOR: helpers

// collect 收一次指標，依名稱回傳每個指標的資料。reader 在記憶體裡，Collect 失敗就是寫錯了，直接 panic。
func collect(reader *sdkmetric.ManualReader) map[string]metricdata.Metrics {
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		panic(err)
	}
	byName := map[string]metricdata.Metrics{}
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			byName[m.Name] = m
		}
	}
	return byName
}

// values 回傳整數指標每個資料點的值，key 是排序過的 "key=value,..." 屬性組合。
// 直方圖的值是落進它的請求數。
func values(m metricdata.Metrics) map[string]int64 {
	out := map[string]int64{}
	add := func(attrs attribute.Set, v int64) { out[attrs.Encoded(attribute.DefaultEncoder())] = v }
	switch d := m.Data.(type) {
	case metricdata.Sum[int64]:
		for _, dp := range d.DataPoints {
			add(dp.Attributes, dp.Value)
		}
	case metricdata.Gauge[int64]:
		for _, dp := range d.DataPoints {
			add(dp.Attributes, dp.Value)
		}
	case metricdata.Histogram[float64]:
		for _, dp := range d.DataPoints {
			add(dp.Attributes, int64(dp.Count))
		}
	default:
		panic("unexpected data type " + m.Name)
	}
	return out
}

// series 回傳指標 name 現有的屬性組合，排序過。
func series(reader *sdkmetric.ManualReader, name string) []string {
	var out []string
	for k := range values(collect(reader)[name]) {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

// ANCHOR_END: helpers
