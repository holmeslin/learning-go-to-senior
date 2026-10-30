package loadshed_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"time"

	loadshed "github.com/holmeslin/learning-go-to-senior/examples/chapter10/07_load_shedding/after"
)

// ANCHOR: example

func Example() {
	entered, release := make(chan struct{}), make(chan struct{})
	slow := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		entered <- struct{}{}
		<-release
	})
	cached := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `["cached"]`)
	})
	s := loadshed.NewShedder(1, 0, 2*time.Second) // 只有 1 個名額
	checkout := s.Handler(loadshed.Critical, slow, nil)
	recommend := s.Handler(loadshed.Optional, slow, cached)

	go checkout.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	<-entered // 名額被第一個請求佔住

	rec := httptest.NewRecorder()
	checkout.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	fmt.Println("checkout:", rec.Code, "Retry-After", rec.Header().Get("Retry-After"))

	rec = httptest.NewRecorder()
	recommend.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	fmt.Println("recommend:", rec.Code, "X-Degraded", rec.Header().Get("X-Degraded"), rec.Body)
	close(release)
	// Output:
	// checkout: 503 Retry-After 2
	// recommend: 200 X-Degraded overload ["cached"]
}

// ANCHOR_END: example
