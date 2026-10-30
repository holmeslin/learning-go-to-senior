package playlist

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

// ANCHOR: dump

// dump 把收到的 span 依父子關係縮排印出來；同一層依結束的順序，span 的事件接在名稱下面。
func dump(rec *tracetest.SpanRecorder) string {
	spans := rec.Ended()
	var sb strings.Builder
	var walk func(parent trace.SpanID, depth int)
	walk = func(parent trace.SpanID, depth int) {
		indent := strings.Repeat("  ", depth)
		for _, s := range spans {
			if s.Parent().SpanID() != parent {
				continue
			}
			fmt.Fprintf(&sb, "%s%s [%s]\n", indent, s.Name(), s.Status().Code)
			for _, e := range s.Events() {
				fmt.Fprintf(&sb, "%s  + %s\n", indent, e.Name)
			}
			walk(s.SpanContext().SpanID(), depth+1)
		}
	}
	walk(trace.SpanID{}, 0)
	return sb.String()
}

// ANCHOR_END: dump

// ANCHOR: example

func Example_trace() {
	rec := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec))

	// 封面服務對 p2 回 503。
	cover := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/p2") {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		io.WriteString(w, "cover.png")
	}))
	defer cover.Close()
	store := &Store{
		Playlists: map[string][]string{"p1": {"t1", "t2", "t3"}, "p2": {"t1"}},
		Titles:    map[string]string{"t1": "a", "t2": "b", "t3": "c"},
	}
	h := NewService(store, cover.Client(), cover.URL).Handler()
	get := func(id string) {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/playlists/"+id, nil))
		fmt.Println(id, "->", w.Code)
	}

	// NewService 沒有 provider 參數，tp 與 recorder 接不上任何東西。
	get("p1")
	fmt.Println("recorded:", len(rec.Ended()))

	// 要看到 span，只能換掉套件變數；這就是測試不能平行的原因。
	defer func(old trace.Tracer) { tracer = old }(tracer)
	tracer = tp.Tracer("playlist")
	get("p1")
	get("p2")
	fmt.Print(dump(rec))
	// Output:
	// p1 -> 200
	// recorded: 0
	// p1 -> 200
	// p2 -> 500
	// get [Unset]
	//   render [Unset]
	//     title [Unset]
	//       normalize [Unset]
	//     title [Unset]
	//       normalize [Unset]
	//     title [Unset]
	//       normalize [Unset]
	// get [Unset]
	//   render [Unset]
	//     title [Unset]
	//       normalize [Unset]
}

// ANCHOR_END: example
