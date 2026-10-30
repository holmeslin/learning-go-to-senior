package voucher_test

import (
	"context"
	"log/slog"
	"os"

	voucher "github.com/holmeslin/learning-go-to-senior/examples/chapter9/01_structured_logging/after"
)

// ANCHOR: notime

// noTime 拿掉 time 欄位，讓 Example 的輸出固定。
func noTime() *slog.HandlerOptions {
	return &slog.HandlerOptions{ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey && len(groups) == 0 {
			return slog.Attr{}
		}
		return a
	}}
}

// ANCHOR_END: notime

// ANCHOR: example

func ExampleRedeemer_Redeem() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, noTime()))
	r := voucher.New(logger, voucher.Voucher{Code: "A1", Cents: 100})
	_ = r.Redeem(context.Background(), "u42", "A1")
	_ = r.Redeem(context.Background(), "u42", "A1")
	// Output:
	// level=INFO msg="voucher redeemed" user_id=u42 voucher.code=A1 voucher.cents=100
	// level=WARN msg="redeem failed" user_id=u42 code=A1 error="voucher already redeemed"
}

// ANCHOR_END: example

// ANCHOR: json

func ExampleRedeemer_Redeem_json() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, noTime()))
	r := voucher.New(logger, voucher.Voucher{Code: "A1", Cents: 100})
	_ = r.Redeem(context.Background(), "u42", "A1")
	// Output:
	// {"level":"INFO","msg":"voucher redeemed","user_id":"u42","voucher":{"code":"A1","cents":100}}
}

// ANCHOR_END: json

// ANCHOR: badkey

func Example_badKey() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, noTime()))
	// 直接寫 logger.Info("redeemed", "user_id", "u42", "cents") 會被 go vet 擋下；
	// 展開 slice 繞過檢查，只為了看看漏掉值時實際會印出什麼。
	args := []any{"user_id", "u42", "cents"}
	logger.Info("redeemed", args...)
	// Output:
	// level=INFO msg=redeemed user_id=u42 !BADKEY=cents
}

// ANCHOR_END: badkey
