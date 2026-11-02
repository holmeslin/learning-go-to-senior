package api_test

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter16/reminder/internal/api"
	"github.com/holmeslin/learning-go-to-senior/examples/chapter16/reminder/internal/store"
)

// ANCHOR: example

// 回應格式：成功是活動編號加四個狀態的數量；錯誤是一個 error 欄位。
func ExampleNewHandler() {
	counter := &fakeCounter{fn: func(eventID int64) (map[store.State]int, error) {
		if eventID == 42 {
			return counts(0, 19980, 15, 5), nil
		}
		return nil, store.ErrNotFound
	}}
	h := api.NewHandler(counter, time.Second, slog.New(slog.DiscardHandler))

	for _, target := range []string{"/v1/events/42/reminder-counts", "/v1/events/43/reminder-counts", "/v1/events/forty-two/reminder-counts"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
		fmt.Printf("GET %s -> %d %s", target, rec.Code, rec.Body.String())
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v1/events/42/reminder-counts", strings.NewReader("{}")))
	fmt.Println("POST ->", rec.Code)
	// Output:
	// GET /v1/events/42/reminder-counts -> 200 {"event_id":42,"counts":{"failed":5,"retrying":15,"sending":0,"sent":19980}}
	// GET /v1/events/43/reminder-counts -> 404 {"error":"event not found"}
	// GET /v1/events/forty-two/reminder-counts -> 400 {"error":"event_id must be a positive integer"}
	// POST -> 405
}

// ANCHOR_END: example
