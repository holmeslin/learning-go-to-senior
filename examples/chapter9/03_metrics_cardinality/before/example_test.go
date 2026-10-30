package httpmetrics_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	httpmetrics "github.com/holmeslin/learning-go-to-senior/examples/chapter9/03_metrics_cardinality/before"
)

// ANCHOR: count

// dataPoints 收一次指標，回傳計數器現在有幾個資料點（每個屬性組合一個）。
func dataPoints(reader *sdkmetric.ManualReader) int {
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		panic(err)
	}
	n := 0
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if sum, ok := m.Data.(metricdata.Sum[int64]); ok {
				n += len(sum.DataPoints)
			}
		}
	}
	return n
}

// ANCHOR_END: count

// ANCHOR: example

func Example() {
	reader := sdkmetric.NewManualReader()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /users/{id}", func(http.ResponseWriter, *http.Request) {})
	h, err := httpmetrics.Middleware(sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)), mux)
	if err != nil {
		panic(err)
	}

	sent := 0
	for _, total := range []int{1, 10, 100, 1000} {
		for ; sent < total; sent++ {
			id := strconv.Itoa(sent)
			req := httptest.NewRequest("GET", "/users/"+id, nil)
			req.Header.Set("X-User-ID", id)
			h.ServeHTTP(httptest.NewRecorder(), req)
		}
		fmt.Printf("%4d requests: %4d data points\n", sent, dataPoints(reader))
	}
	// Output:
	//    1 requests:    1 data points
	//   10 requests:   10 data points
	//  100 requests:  100 data points
	// 1000 requests: 1000 data points
}

// ANCHOR_END: example
