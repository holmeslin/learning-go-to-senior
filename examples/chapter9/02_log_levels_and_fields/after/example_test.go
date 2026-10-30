package billing_test

import (
	"log/slog"
	"net/http/httptest"
	"os"

	billing "github.com/holmeslin/learning-go-to-senior/examples/chapter9/02_log_levels_and_fields/after"
)

// ANCHOR: drop

// dropVolatile 拿掉每次執行都不同的 time 與 duration_ms，讓 Example 的輸出固定。
func dropVolatile(groups []string, a slog.Attr) slog.Attr {
	if len(groups) == 0 && (a.Key == slog.TimeKey || a.Key == billing.KeyDuration) {
		return slog.Attr{}
	}
	return a
}

// ANCHOR_END: drop

// ANCHOR: request

func ExampleNewHandler() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{ReplaceAttr: dropVolatile}))
	h := billing.NewHandler(&billing.Service{Store: &billing.Store{Down: true}}, logger)

	// 和 before 一樣：一個請求、一次失敗。
	req := httptest.NewRequest("POST", "/invoices/inv-7/pay", nil)
	req.Header.Set("X-Request-ID", "r-1")
	req.Header.Set("X-User-ID", "u-42")
	h.ServeHTTP(httptest.NewRecorder(), req)
	// Output:
	// level=ERROR msg="pay invoice failed" request_id=r-1 user_id=u-42 invoice_id=inv-7 error="pay invoice inv-7: connection refused"
}

// ANCHOR_END: request

// ANCHOR: levelvar

func ExampleNewHandler_levelVar() {
	var level slog.LevelVar // 零值是 Info
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: &level, ReplaceAttr: dropVolatile}))
	h := billing.NewHandler(&billing.Service{Store: &billing.Store{}}, logger)

	pay := func(requestID string) {
		req := httptest.NewRequest("POST", "/invoices/inv-7/pay", nil)
		req.Header.Set("X-Request-ID", requestID)
		req.Header.Set("X-User-ID", "u-42")
		h.ServeHTTP(httptest.NewRecorder(), req)
	}
	pay("r-1")
	level.Set(slog.LevelDebug) // 不用重新部署，下一個請求就生效
	pay("r-2")
	// Output:
	// level=INFO msg="invoice paid" request_id=r-1 user_id=u-42 invoice_id=inv-7
	// level=DEBUG msg="charge invoice" request_id=r-2 user_id=u-42 invoice_id=inv-7
	// level=INFO msg="invoice paid" request_id=r-2 user_id=u-42 invoice_id=inv-7
}

// ANCHOR_END: levelvar

// ANCHOR: importlog

func ExampleImportInvoices() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{ReplaceAttr: dropVolatile}))
	billing.ImportInvoices(logger, []string{"inv-1", "inv-2", "", "inv-4"})
	// Output:
	// level=WARN msg="import finished" imported=3 skipped=1
}

// ANCHOR_END: importlog
