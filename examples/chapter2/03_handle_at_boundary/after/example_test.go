package taskapi_test

import (
	"fmt"
	"log"
	"net/http/httptest"
	"os"

	taskapi "github.com/holmeslin/learning-go-to-senior/examples/chapter2/03_handle_at_boundary/after"
)

// ANCHOR: example

func ExampleNewHandler() {
	logger := log.New(os.Stdout, "", 0)
	svc := &taskapi.Service{Store: &taskapi.Store{Down: true}}
	h := taskapi.NewHandler(svc, logger)

	// 同樣一個請求、一次失敗：只在 handler 記錄一次。
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("POST", "/tasks/7/complete", nil))
	fmt.Println(rec.Code)
	// Output:
	// handler: POST /tasks/7/complete: complete: save task 7: connection refused
	// 500
}

// ANCHOR_END: example

func ExampleService_Complete() {
	svc := &taskapi.Service{Store: &taskapi.Store{Down: true}}

	// 中間層沒有 logger，只把加上上下文的錯誤交還給呼叫端。
	fmt.Println(svc.Complete(7))
	// Output: complete: save task 7: connection refused
}

// ANCHOR: worker

func ExampleRunWorker() {
	logger := log.New(os.Stdout, "", 0)
	svc := &taskapi.Service{Store: &taskapi.Store{Down: true}}

	ids := make(chan int, 2)
	ids <- 7
	ids <- 8
	close(ids)
	taskapi.RunWorker(ids, svc, logger)
	// Output:
	// worker: task 7: complete: save task 7: connection refused
	// worker: task 8: complete: save task 8: connection refused
}

// ANCHOR_END: worker
