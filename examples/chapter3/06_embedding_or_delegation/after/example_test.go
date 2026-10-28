package visits_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"

	visits "github.com/holmeslin/learning-go-to-senior/examples/chapter3/06_embedding_or_delegation/after"
)

// ANCHOR: counter

func ExampleCounter() {
	var c visits.Counter
	_, isLocker := any(&c).(sync.Locker)
	fmt.Println("is a sync.Locker:", isLocker)
	// Output: is a sync.Locker: false
}

// ANCHOR_END: counter

// ANCHOR: audited

func ExampleAudited() {
	var c visits.Counter
	log := func(msg string) { fmt.Println("audit:", msg) }

	// Audited 得自己寫出 Store 的每個方法，少一個就不能當 Store 用。
	var s visits.Store = visits.NewAudited(&c, log)
	s.Incr("/home")
	s.Reset("/home")
	fmt.Println("count after reset:", s.Count("/home"))
	// Output:
	// audit: incr /home
	// audit: reset /home
	// count after reset: 0
}

// ANCHOR_END: audited

// ANCHOR: flusher

// statusRecorder 嵌入 http.ResponseWriter，只想多記一個狀態碼。
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// Unwrap 讓 http.NewResponseController 找得到底下的 ResponseWriter。
func (r *statusRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }

func Example_wrapResponseWriter() {
	rec := httptest.NewRecorder() // 它同時是 http.ResponseWriter 和 http.Flusher
	_, ok := http.ResponseWriter(rec).(http.Flusher)
	fmt.Println("original is an http.Flusher:", ok)

	sr := &statusRecorder{ResponseWriter: rec}
	var w http.ResponseWriter = sr
	w.WriteHeader(http.StatusAccepted)
	fmt.Println("recorded status:", sr.status)

	_, ok = w.(http.Flusher)
	fmt.Println("wrapper is an http.Flusher:", ok)

	err := http.NewResponseController(w).Flush()
	fmt.Println("flush error:", err, "flushed:", rec.Flushed)
	// Output:
	// original is an http.Flusher: true
	// recorded status: 202
	// wrapper is an http.Flusher: false
	// flush error: <nil> flushed: true
}

// ANCHOR_END: flusher
