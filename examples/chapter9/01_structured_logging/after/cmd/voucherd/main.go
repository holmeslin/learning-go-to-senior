// Command voucherd 示範在進入點依環境選 handler，再把 logger 交給函式庫。
package main

import (
	"context"
	"log/slog"
	"os"

	voucher "github.com/holmeslin/learning-go-to-senior/examples/chapter9/01_structured_logging/after"
)

// ANCHOR: main

func main() {
	var h slog.Handler = slog.NewTextHandler(os.Stderr, nil) // 本機：給人看
	if os.Getenv("APP_ENV") == "production" {
		h = slog.NewJSONHandler(os.Stderr, nil) // 正式環境：給日誌收集系統解析
	}
	logger := slog.New(h)

	r := voucher.New(logger, voucher.Voucher{Code: "A1", Cents: 100})
	r.Redeem(context.Background(), "u42", "A1")
}

// ANCHOR_END: main
