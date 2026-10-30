package httpmetrics_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"

	httpmetrics "github.com/holmeslin/learning-go-to-senior/examples/chapter9/03_metrics_cardinality/after"
)

// ANCHOR: example

func ExampleMiddleware() {
	reader := sdkmetric.NewManualReader()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /users/{id}", func(http.ResponseWriter, *http.Request) {})
	h, err := httpmetrics.Middleware(sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)), mux)
	if err != nil {
		panic(err)
	}

	paths := []string{"/robots.txt"} // 對不上任何路由
	for i := range 1000 {
		paths = append(paths, "/users/"+strconv.Itoa(i))
	}
	for _, p := range paths {
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", p, nil))
	}

	for _, s := range series(reader, "http.server.requests") {
		fmt.Println(s)
	}
	// Output:
	// http.request.method=GET,http.response.status_class=2xx,http.route=/users/{id}
	// http.request.method=GET,http.response.status_class=4xx,http.route=unmatched
}

// ANCHOR_END: example
