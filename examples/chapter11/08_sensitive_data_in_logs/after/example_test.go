package wallet_test

import (
	"log/slog"
	"os"

	wallet "github.com/holmeslin/learning-go-to-senior/examples/chapter11/08_sensitive_data_in_logs/after"
)

// ANCHOR: example

// redactNoTime 先套用 wallet.Redact，再拿掉每次都不同的 time，讓輸出固定。
func redactNoTime(groups []string, a slog.Attr) slog.Attr {
	if len(groups) == 0 && a.Key == slog.TimeKey {
		return slog.Attr{}
	}
	return wallet.Redact(groups, a)
}

func ExampleNewHandler() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{ReplaceAttr: redactNoTime}))
	bind(wallet.NewHandler(logger)) // 和 before 送一模一樣的請求
	// Output:
	// level=INFO msg="bind card" request.user_id=u-42 request.password=[REDACTED] request.card.last4=3456 headers.Authorization=[REDACTED] headers.Content-Type=application/json
}

// ANCHOR_END: example

// ANCHOR: opaque

func ExampleRedact_opaqueValue() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{ReplaceAttr: redactNoTime}))
	type rawRequest struct{ Password string }
	logger.Info("by key", slog.String("password", "example-not-a-real-password"))
	logger.Info("by struct", slog.Any("req", rawRequest{Password: "example-not-a-real-password"}))
	// Output:
	// level=INFO msg="by key" password=[REDACTED]
	// level=INFO msg="by struct" req={Password:example-not-a-real-password}
}

// ANCHOR_END: opaque
