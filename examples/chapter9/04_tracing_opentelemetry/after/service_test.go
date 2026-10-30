package playlist_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

	playlist "github.com/holmeslin/learning-go-to-senior/examples/chapter9/04_tracing_opentelemetry/after"
)

// ANCHOR: fixture

// newHandler 建一個 Service：provider 由參數傳入，封面服務固定回 coverStatus。
func newHandler(t *testing.T, tp trace.TracerProvider, coverStatus int) http.Handler {
	t.Helper()
	cover := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(coverStatus)
		io.WriteString(w, "cover.png")
	}))
	t.Cleanup(cover.Close)
	store := &playlist.Store{
		Playlists: map[string][]string{
			"p1": {"t1"},
			"p3": {"t1", "t2", "t3"},
			"p5": {"t1", "t2", "t3", "t4", "t5"},
		},
		Titles: map[string]string{"t1": "a", "t2": "b", "t3": "c", "t4": "d", "t5": "e"},
	}
	return playlist.NewService(tp, store, cover.Client(), cover.URL).Handler()
}

// newProvider 每個測試建自己的 provider 與 recorder，不碰全域設定，所以可以平行。
func newProvider(opts ...sdktrace.TracerProviderOption) (*sdktrace.TracerProvider, *tracetest.SpanRecorder) {
	rec := tracetest.NewSpanRecorder()
	opts = append(opts, sdktrace.WithSpanProcessor(rec))
	return sdktrace.NewTracerProvider(opts...), rec
}

// ANCHOR_END: fixture

// get 對 h 送出 GET，回傳狀態碼。
func get(h http.Handler, path string) int {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec.Code
}

// attr 取出 span 的屬性。
func attr(s sdktrace.ReadOnlySpan, key string) attribute.Value {
	set := attribute.NewSet(s.Attributes()...)
	v, _ := set.Value(attribute.Key(key))
	return v
}

// ANCHOR: siblings

func TestNPlusOneIsSiblingSpans(t *testing.T) {
	t.Parallel()
	tests := []struct {
		id     string
		tracks []string
	}{
		{"p1", []string{"t1"}},
		{"p3", []string{"t1", "t2", "t3"}},
		{"p5", []string{"t1", "t2", "t3", "t4", "t5"}},
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			t.Parallel()
			tp, rec := newProvider()
			get(newHandler(t, tp, http.StatusOK), "/playlists/"+tt.id)

			spans := rec.Ended()
			root := spans[len(spans)-1] // 請求 span 最後結束
			if root.Name() != "GET /playlists/{id}" || root.Parent().IsValid() {
				t.Fatalf("last span = %q (has parent: %v), want the root request span", root.Name(), root.Parent().IsValid())
			}
			var names, tracks []string
			for _, s := range spans[:len(spans)-1] {
				if s.Parent().SpanID() != root.SpanContext().SpanID() {
					t.Errorf("%s: parent is not the request span", s.Name())
				}
				names = append(names, s.Name())
				if s.Name() == "SELECT tracks" {
					tracks = append(tracks, attr(s, "track.id").AsString())
				}
			}
			if !slices.Equal(tracks, tt.tracks) {
				t.Errorf("SELECT tracks spans = %v, want one per track %v", tracks, tt.tracks)
			}
			if got, want := len(names), len(tt.tracks)+2; got != want {
				t.Errorf("child spans = %v, want %d: one per track, one playlist query, one cover call", names, want)
			}
		})
	}
}

// ANCHOR_END: siblings

// ANCHOR: status

func TestStatusAndErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		path        string
		coverStatus int
		wantCode    int
		wantErrored []string
	}{
		{"ok", "/playlists/p3", 200, 200, nil},
		{"unknown playlist", "/playlists/nope", 200, 404, nil},
		{"cover service down", "/playlists/p3", 503, 500, []string{"GET /covers/{id}", "GET /playlists/{id}"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tp, rec := newProvider()
			if got := get(newHandler(t, tp, tt.coverStatus), tt.path); got != tt.wantCode {
				t.Fatalf("status = %d, want %d", got, tt.wantCode)
			}

			var errored []string
			for _, s := range rec.Ended() {
				if s.Name() == "GET /playlists/{id}" {
					if got := attr(s, "http.response.status_code").AsInt64(); got != int64(tt.wantCode) {
						t.Errorf("status_code attribute = %d, want %d", got, tt.wantCode)
					}
				}
				if s.Status().Code != codes.Error {
					continue
				}
				errored = append(errored, s.Name())
				hasException := slices.ContainsFunc(s.Events(), func(e sdktrace.Event) bool { return e.Name == "exception" })
				if !hasException {
					t.Errorf("%s: status is Error but no exception event", s.Name())
				}
			}
			slices.Sort(errored)
			if !slices.Equal(errored, tt.wantErrored) {
				t.Errorf("spans with Error status = %v, want %v", errored, tt.wantErrored)
			}
		})
	}
}

// ANCHOR_END: status

// ANCHOR: sampling

func TestSamplingIsProviderConfig(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		ratio float64
		want  int
	}{
		{"sample everything", 1, 6},
		{"sample nothing", 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			sampler := sdktrace.ParentBased(sdktrace.TraceIDRatioBased(tt.ratio))
			tp, rec := newProvider(sdktrace.WithSampler(sampler))
			if got := get(newHandler(t, tp, http.StatusOK), "/playlists/p3"); got != http.StatusOK {
				t.Fatalf("status = %d, want 200", got)
			}
			if got := len(rec.Ended()); got != tt.want {
				t.Errorf("recorded spans = %d, want %d", got, tt.want)
			}
		})
	}
}

// ANCHOR_END: sampling

func TestNoopProvider(t *testing.T) {
	t.Parallel()
	if got := get(newHandler(t, noop.NewTracerProvider(), http.StatusOK), "/playlists/p3"); got != http.StatusOK {
		t.Errorf("status = %d, want 200", got)
	}
}
