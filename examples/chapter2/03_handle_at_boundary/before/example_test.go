package taskapi_test

import (
	"fmt"
	"log"
	"net/http/httptest"
	"os"

	taskapi "github.com/holmeslin/learning-go-to-senior/examples/chapter2/03_handle_at_boundary/before"
)

// ANCHOR: example

func ExampleNewHandler() {
	logger := log.New(os.Stdout, "", 0)
	store := &taskapi.Store{Log: logger, Down: true}
	svc := &taskapi.Service{Log: logger, Store: store}
	h := taskapi.NewHandler(svc, logger)

	// 一個請求、一次失敗。
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("POST", "/tasks/7/complete", nil))
	fmt.Println(rec.Code)
	// Output:
	// store: save task 7: connection refused
	// service: complete: save task 7: connection refused
	// handler: POST /tasks/7/complete: complete: save task 7: connection refused
	// 500
}

// ANCHOR_END: example
