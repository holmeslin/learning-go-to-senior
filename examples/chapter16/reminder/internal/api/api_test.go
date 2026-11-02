package api_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter16/reminder/internal/api"
	"github.com/holmeslin/learning-go-to-senior/examples/chapter16/reminder/internal/store"
)

// ANCHOR: fake

// fakeCounter 是假的 api.Counter：每次呼叫記下收到的活動編號與 ctx 的期限，結果由 fn 決定。
type fakeCounter struct {
	fn func(eventID int64) (map[store.State]int, error)

	calls       []int64
	hadDeadline bool
}

func (f *fakeCounter) CountByState(ctx context.Context, eventID int64) (map[store.State]int, error) {
	f.calls = append(f.calls, eventID)
	_, f.hadDeadline = ctx.Deadline()
	return f.fn(eventID)
}

// ANCHOR_END: fake

func counts(sending, sent, retrying, failed int) map[store.State]int {
	return map[store.State]int{
		store.StateSending: sending, store.StateSent: sent, store.StateRetrying: retrying, store.StateFailed: failed,
	}
}

// do 對 handler 送出請求，回傳狀態碼、標頭與本文。
func do(h http.Handler, method, target string) (int, http.Header, string) {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, target, nil))
	body, _ := io.ReadAll(rec.Result().Body)
	return rec.Code, rec.Result().Header, string(body)
}

func newHandler(c api.Counter) http.Handler {
	return api.NewHandler(c, time.Second, slog.New(slog.DiscardHandler))
}

// ANCHOR: validation

// 活動編號是正整數的標準寫法；其他寫法一律 400，而且不會去查資料。
func TestHandlerRejectsInvalidEventIDs(t *testing.T) {
	t.Parallel()
	for _, id := range []string{
		"abc", "0", "-1", "1.5", "1e3", "0x10",
		"+5",                   // ParseInt 接受，但同一場活動會有兩個網址
		"007",                  // 同上
		"9223372036854775808",  // 超過 int64
		"99999999999999999999", // 遠超過 int64
	} {
		t.Run(id, func(t *testing.T) {
			t.Parallel()
			fake := &fakeCounter{fn: func(int64) (map[store.State]int, error) { return counts(0, 0, 0, 0), nil }}
			code, _, body := do(newHandler(fake), http.MethodGet, "/v1/events/"+id+"/reminder-counts")
			if code != http.StatusBadRequest {
				t.Errorf("GET %s = %d %s, want 400", id, code, body)
			}
			if len(fake.calls) != 0 {
				t.Errorf("counter called with %v, want no query for an invalid id", fake.calls)
			}
		})
	}
}

// ANCHOR_END: validation

// ANCHOR: status

// 資料層的錯誤對應到狀態碼：沒有這場活動 404、查詢逾時 503、其他 500；500 的本文不帶內部的錯誤訊息。
func TestHandlerMapsErrorsToStatusCodes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"unknown event", fmt.Errorf("count deliveries of event 7: %w", store.ErrNotFound), http.StatusNotFound},
		{"query timeout", fmt.Errorf("count deliveries: %w", context.DeadlineExceeded), http.StatusServiceUnavailable},
		{"anything else", errors.New("disk I/O error at /var/lib/reminder.db"), http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			fake := &fakeCounter{fn: func(int64) (map[store.State]int, error) { return nil, tt.err }}
			code, header, body := do(newHandler(fake), http.MethodGet, "/v1/events/7/reminder-counts")
			if code != tt.want {
				t.Errorf("status = %d, want %d", code, tt.want)
			}
			if got := header.Get("Content-Type"); got != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", got)
			}
			if strings.Contains(body, "/var/lib") || strings.Contains(body, "disk") {
				t.Errorf("body = %s, want no internal error detail", body)
			}
		})
	}
}

// ANCHOR_END: status

// ANCHOR: readonly

// 沒有寫入的操作：改變資料的方法一律 405，Allow 只列讀取的方法，而且不會去查資料。
func TestHandlerOffersNoWriteOperations(t *testing.T) {
	t.Parallel()
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			t.Parallel()
			fake := &fakeCounter{fn: func(int64) (map[store.State]int, error) { return counts(0, 0, 0, 0), nil }}
			code, header, _ := do(newHandler(fake), method, "/v1/events/7/reminder-counts")
			if code != http.StatusMethodNotAllowed {
				t.Errorf("%s = %d, want 405", method, code)
			}
			if allow := strings.Split(header.Get("Allow"), ", "); !slices.Equal(allow, []string{"GET", "HEAD"}) {
				t.Errorf("Allow = %v, want GET and HEAD only", allow)
			}
			if len(fake.calls) != 0 {
				t.Errorf("counter called with %v, want no query", fake.calls)
			}
		})
	}
}

// ANCHOR_END: readonly

// 查詢帶著期限：資料庫卡住時，請求不會跟著卡住。路徑以外的東西（診斷端點）不在這個 handler 上。
func TestHandlerSetsAQueryDeadlineAndServesOnlyItsRoute(t *testing.T) {
	t.Parallel()
	fake := &fakeCounter{fn: func(int64) (map[store.State]int, error) { return counts(0, 1, 0, 0), nil }}
	h := newHandler(fake)
	if code, _, body := do(h, http.MethodGet, "/v1/events/7/reminder-counts"); code != http.StatusOK {
		t.Fatalf("GET = %d %s, want 200", code, body)
	}
	if !fake.hadDeadline {
		t.Error("counter ctx has no deadline, want the query timeout")
	}
	for _, path := range []string{"/debug/pprof/", "/healthz", "/v1/events/7", "/v1/events/7/reminder-counts/extra"} {
		if code, _, _ := do(h, http.MethodGet, path); code != http.StatusNotFound {
			t.Errorf("GET %s = %d, want 404", path, code)
		}
	}
}
