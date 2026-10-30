package billing_test

import (
	"log/slog"
	"net/http/httptest"
	"os"

	billing "github.com/holmeslin/learning-go-to-senior/examples/chapter9/02_log_levels_and_fields/before"
)

// newLogger 回傳寫到標準輸出的文字 logger，拿掉 time 讓輸出固定。
func newLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if len(groups) == 0 && a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	}))
}

// ANCHOR: request

func ExampleNewHandler() {
	logger := newLogger()
	store := &billing.Store{Down: true, Logger: logger}
	h := billing.NewHandler(&billing.Service{Store: store, Logger: logger}, logger)

	// 一個請求、一次失敗。
	req := httptest.NewRequest("POST", "/invoices/inv-7/pay", nil)
	req.Header.Set("X-User-ID", "u-42")
	h.ServeHTTP(httptest.NewRecorder(), req)
	// Output:
	// level=ERROR msg="charge failed" userId=u-42 err="connection refused"
	// level=ERROR msg="pay failed" user_id=u-42 invoice_id=inv-7 error="connection refused"
	// level=ERROR msg="request failed" uid=u-42 inv=inv-7 err="pay invoice inv-7: connection refused"
}

// ANCHOR_END: request

// ANCHOR: importlog

func ExampleImportInvoices() {
	billing.ImportInvoices(newLogger(), []string{"inv-1", "inv-2", "", "inv-4"})
	// Output:
	// level=INFO msg=imported invoiceId=inv-1
	// level=INFO msg=imported invoiceId=inv-2
	// level=INFO msg="empty row"
	// level=INFO msg=imported invoiceId=inv-4
}

// ANCHOR_END: importlog
