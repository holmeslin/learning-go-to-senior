package diag_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter16/reminder/internal/buildinfo"
	"github.com/holmeslin/learning-go-to-senior/examples/chapter16/reminder/internal/diag"
)

// fakeSnapshot 是假的 diag.Snapshotter。
type fakeSnapshot struct {
	text string
	err  error
}

func (f fakeSnapshot) Snapshot(_ context.Context, w io.Writer) error {
	if f.err != nil {
		return f.err
	}
	_, err := io.WriteString(w, f.text)
	return err
}

func request(h http.Handler, method, target string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, target, nil))
	return rec
}

// ANCHOR: ready

// readiness 的轉換：啟動完成前未就緒、啟動完成後就緒、關機開始後又回到未就緒；
// 健康檢查（行程還活著）不受影響，pprof 在關機期間也還能用。
func TestReadyzFollowsReadiness(t *testing.T) {
	t.Parallel()
	ready := &diag.Readiness{}
	h := diag.NewHandler(ready, fakeSnapshot{}, buildinfo.Info{})

	steps := []struct {
		name  string
		set   func()
		ready int
	}{
		{"before startup finishes", func() {}, http.StatusServiceUnavailable},
		{"started", func() { ready.Set(true) }, http.StatusOK},
		{"shutdown begins", func() { ready.Set(false) }, http.StatusServiceUnavailable},
	}
	for _, step := range steps {
		step.set()
		if got := request(h, http.MethodGet, "/readyz").Code; got != step.ready {
			t.Errorf("%s: GET /readyz = %d, want %d", step.name, got, step.ready)
		}
		if got := request(h, http.MethodGet, "/healthz").Code; got != http.StatusOK {
			t.Errorf("%s: GET /healthz = %d, want 200", step.name, got)
		}
		if got := request(h, http.MethodGet, "/debug/pprof/goroutine?debug=1").Code; got != http.StatusOK {
			t.Errorf("%s: GET /debug/pprof/goroutine = %d, want 200", step.name, got)
		}
	}
}

// ANCHOR_END: ready

// ANCHOR: routes

// 診斷端點是明確註冊的一份清單，在自己的 ServeMux 上：pprof 只有具名的 profile，
// 沒有 cmdline（命令列可能帶著秘密）；沒有業務路由；不接受 GET 以外的方法。
func TestHandlerServesOnlyTheChosenEndpoints(t *testing.T) {
	t.Parallel()
	h := diag.NewHandler(&diag.Readiness{}, fakeSnapshot{text: "x 1\n"}, buildinfo.Info{})
	tests := []struct {
		method, target string
		want           int
	}{
		{http.MethodGet, "/healthz", http.StatusOK},
		{http.MethodGet, "/version", http.StatusOK},
		{http.MethodGet, "/debug/pprof/", http.StatusOK},
		{http.MethodGet, "/debug/pprof/heap", http.StatusOK},
		{http.MethodGet, "/debug/pprof/symbol", http.StatusOK},
		{http.MethodGet, "/debug/metrics", http.StatusOK},
		{http.MethodGet, "/debug/pprof/cmdline", http.StatusNotFound},
		{http.MethodGet, "/v1/events/1/reminder-counts", http.StatusNotFound},
		{http.MethodPost, "/readyz", http.StatusMethodNotAllowed},
		{http.MethodPost, "/version", http.StatusMethodNotAllowed},
		{http.MethodPost, "/debug/pprof/", http.StatusMethodNotAllowed},
	}
	for _, tt := range tests {
		if got := request(h, tt.method, tt.target).Code; got != tt.want {
			t.Errorf("%s %s = %d, want %d", tt.method, tt.target, got, tt.want)
		}
	}
}

// ANCHOR_END: routes

func TestMetricsEndpointWritesTheSnapshot(t *testing.T) {
	t.Parallel()
	rec := request(diag.NewHandler(&diag.Readiness{}, fakeSnapshot{text: "reminder.deliveries{result=\"sent\"} 3\n"}, buildinfo.Info{}), http.MethodGet, "/debug/metrics")
	if rec.Code != http.StatusOK || !strings.HasPrefix(rec.Header().Get("Content-Type"), "text/plain") {
		t.Errorf("GET /debug/metrics = %d %q, want 200 text/plain", rec.Code, rec.Header().Get("Content-Type"))
	}
	if got := rec.Body.String(); got != "reminder.deliveries{result=\"sent\"} 3\n" {
		t.Errorf("body = %q, want the snapshot", got)
	}
}

func TestMetricsEndpointHidesSnapshotErrors(t *testing.T) {
	t.Parallel()
	rec := request(diag.NewHandler(&diag.Readiness{}, fakeSnapshot{err: errors.New("collect failed: secret detail")}, buildinfo.Info{}), http.MethodGet, "/debug/metrics")
	if rec.Code != http.StatusInternalServerError || strings.Contains(rec.Body.String(), "secret") {
		t.Errorf("GET /debug/metrics = %d %q, want 500 without the error detail", rec.Code, rec.Body.String())
	}
}

// 版本端點把 build 原樣回成 JSON；欄位名稱是給 curl 與值班的人看的契約。
func TestVersionEndpointServesTheBuildInfo(t *testing.T) {
	t.Parallel()
	info := buildinfo.Info{Version: "v1.5.0", Revision: "7a0c240b07b4", Time: "2030-06-10T08:00:00Z", GoVersion: "go1.27.0"}
	rec := request(diag.NewHandler(&diag.Readiness{}, fakeSnapshot{}, info), http.MethodGet, "/version")
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "application/json" {
		t.Errorf("GET /version = %d %q, want 200 application/json", rec.Code, rec.Header().Get("Content-Type"))
	}
	const want = `{"version":"v1.5.0","revision":"7a0c240b07b4","time":"2030-06-10T08:00:00Z","modified":false,"go_version":"go1.27.0"}` + "\n"
	if got := rec.Body.String(); got != want {
		t.Errorf("body = %q, want %q", got, want)
	}
}
