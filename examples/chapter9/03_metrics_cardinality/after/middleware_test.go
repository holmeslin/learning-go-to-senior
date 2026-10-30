package httpmetrics_test

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"testing"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	httpmetrics "github.com/holmeslin/learning-go-to-senior/examples/chapter9/03_metrics_cardinality/after"
)

// ANCHOR: bounded

// newServer 回傳包好中介層的 handler，以及收這個 handler 指標的 reader。
func newServer(t *testing.T) (http.Handler, *sdkmetric.ManualReader) {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /users/{id}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("id") == "missing" {
			http.NotFound(w, r)
		}
	})
	mux.HandleFunc("POST /users", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
	reader := sdkmetric.NewManualReader()
	h, err := httpmetrics.Middleware(sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)), mux)
	if err != nil {
		t.Fatal(err)
	}
	return h, reader
}

// send 送一個請求給 h，使用者代號放在 X-User-ID 標頭。
func send(h http.Handler, method, target, userID string) {
	req := httptest.NewRequest(method, target, nil)
	req.Header.Set("X-User-ID", userID)
	h.ServeHTTP(httptest.NewRecorder(), req)
}

func TestSeriesDoNotGrowWithRequests(t *testing.T) {
	t.Parallel()
	h, reader := newServer(t)

	for i := range 500 {
		id := strconv.Itoa(i)
		send(h, "GET", "/users/"+id, id) // 每個請求都是不同的路徑與使用者
	}
	send(h, "GET", "/users/missing", "7")
	send(h, "POST", "/users", "7")
	for i := range 100 {
		send(h, "GET", "/wp-admin/"+strconv.Itoa(i), "") // 掃描程式探的路徑，沒有一條對得上路由
	}

	want := []string{
		"http.request.method=GET,http.response.status_class=2xx,http.route=/users/{id}",
		"http.request.method=GET,http.response.status_class=4xx,http.route=/users/{id}",
		"http.request.method=GET,http.response.status_class=4xx,http.route=unmatched",
		"http.request.method=POST,http.response.status_class=2xx,http.route=/users",
	}
	if got := series(reader, "http.server.requests"); !slices.Equal(got, want) {
		t.Errorf("series = %q, want %q", got, want)
	}
}

// ANCHOR_END: bounded

func TestHistogramSharesAttributes(t *testing.T) {
	t.Parallel()
	h, reader := newServer(t)
	send(h, "GET", "/users/1", "1")
	send(h, "GET", "/users/missing", "1")
	send(h, "POST", "/users", "1")

	counter := series(reader, "http.server.requests")
	histogram := series(reader, "http.server.request.duration")
	if len(counter) != 3 || !slices.Equal(counter, histogram) {
		t.Errorf("counter series = %q, histogram series = %q, want the same three", counter, histogram)
	}
}

// ANCHOR: method

func TestUnknownMethodsShareOneSeries(t *testing.T) {
	t.Parallel()
	h, reader := newServer(t)
	for _, m := range []string{"BREW", "PURGE", "PROPFIND"} {
		send(h, m, "/users/1", "1") // 路徑對得上，方法沒有註冊：mux 回 405
	}

	want := []string{"http.request.method=_OTHER,http.response.status_class=4xx,http.route=unmatched"}
	if got := series(reader, "http.server.requests"); !slices.Equal(got, want) {
		t.Errorf("series = %q, want %q", got, want)
	}
}

// ANCHOR_END: method

func TestDurationBucketsAreInSeconds(t *testing.T) {
	t.Parallel()
	h, reader := newServer(t)
	send(h, "GET", "/users/1", "1")

	hist, ok := collect(reader)["http.server.request.duration"].Data.(metricdata.Histogram[float64])
	if !ok || len(hist.DataPoints) != 1 {
		t.Fatalf("duration data = %v, want one histogram data point", hist)
	}
	// SDK 預設的邊界是 0 到 10000（為毫秒設計）；用秒記耗時，邊界要涵蓋 0.1 秒這種常見的門檻，最大的邊界也不該高到 60 以上。
	bounds := hist.DataPoints[0].Bounds
	if !slices.Contains(bounds, 0.1) || bounds[len(bounds)-1] > 60 {
		t.Errorf("bounds = %v, want boundaries in seconds (including 0.1, none above 60)", bounds)
	}
}

func TestStatusClassDefaultsTo2xx(t *testing.T) {
	t.Parallel()
	h, reader := newServer(t)
	send(h, "GET", "/users/1", "1") // handler 沒有呼叫 WriteHeader，net/http 回 200

	want := []string{"http.request.method=GET,http.response.status_class=2xx,http.route=/users/{id}"}
	if got := series(reader, "http.server.requests"); !slices.Equal(got, want) {
		t.Errorf("series = %q, want %q", got, want)
	}
}
