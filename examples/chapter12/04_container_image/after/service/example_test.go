package service_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter12/04_container_image/after/service"
)

// ANCHOR: example

func Example() {
	h := service.NewHandler()
	for _, target := range []string{"/livez", "/hello?name=gopher"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
		fmt.Printf("%s %d %q\n", target, rec.Code, rec.Body.String())
	}
	// Output:
	// /livez 200 "ok"
	// /hello?name=gopher 200 "hello, gopher\n"
}

// ANCHOR_END: example
