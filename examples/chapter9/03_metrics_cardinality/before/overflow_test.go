package httpmetrics_test

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"

	httpmetrics "github.com/holmeslin/learning-go-to-senior/examples/chapter9/03_metrics_cardinality/before"
)

// ANCHOR: overflow

// SDK 有資料點的上限（預設也是 2000）：超過的組合被併進一個 otel.metric.overflow 資料點。
func TestSDKCapsSeries(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	h, err := httpmetrics.Middleware(
		sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader), sdkmetric.WithCardinalityLimit(2000)),
		http.NotFoundHandler())
	if err != nil {
		t.Fatal(err)
	}
	for i := range 3000 {
		req := httptest.NewRequest("GET", "/users/"+strconv.Itoa(i), nil)
		h.ServeHTTP(httptest.NewRecorder(), req)
	}
	if got := dataPoints(reader); got != 2000 {
		t.Errorf("data points = %d, want 2000 (limit, including the overflow point)", got)
	}
}

// ANCHOR_END: overflow
