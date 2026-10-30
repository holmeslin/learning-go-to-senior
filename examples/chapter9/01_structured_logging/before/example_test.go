package voucher_test

import (
	"bytes"
	"fmt"
	"log/slog"
	"strings"

	voucher "github.com/holmeslin/learning-go-to-senior/examples/chapter9/01_structured_logging/before"
)

// ANCHOR: example

func ExampleRedeemer_Redeem() {
	var buf bytes.Buffer
	opts := &slog.HandlerOptions{ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey && len(groups) == 0 {
			return slog.Attr{}
		}
		return a
	}}
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, opts)))
	defer slog.SetDefault(old)

	r := voucher.New(voucher.Voucher{Code: "A1", Cents: 100})
	r.Redeem("u4", "ZZ")
	r.Redeem("u42", "A1")
	r.Redeem("u42", "A1")

	// 「找 u4 的所有失敗」：欄位都在訊息裡，只能比對字串。
	for line := range strings.Lines(buf.String()) {
		if strings.Contains(line, `"WARN"`) && strings.Contains(line, "user u4") {
			fmt.Print(line)
		}
	}
	// Output:
	// {"level":"WARN","msg":"redeem failed: user u4 code ZZ: unknown voucher"}
	// {"level":"WARN","msg":"redeem failed: user u42 code A1: voucher already redeemed"}
}

// ANCHOR_END: example
