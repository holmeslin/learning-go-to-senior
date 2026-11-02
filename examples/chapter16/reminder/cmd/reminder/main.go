// Command reminder 是提醒服務的行程入口：讀環境變數、處理 signal、組裝，其他的事都交給 reminder.Service。
//
// 執行時至少要設 REMINDER_PROVIDER_URL；送出的開關（REMINDER_SEND_ENABLED）預設關閉。
// 全部的設定名稱與預設值見 config.Config。內部 API 預設在 :8080，診斷端點（健康檢查、readiness、
// /debug/pprof、/debug/metrics）預設在 127.0.0.1:6060，只聽 loopback。
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter16/reminder"
	"github.com/holmeslin/learning-go-to-senior/examples/chapter16/reminder/internal/config"
)

// ANCHOR: main

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	if err := run(log); err != nil {
		log.Error("reminder service stopped", "error", err)
		os.Exit(1)
	}
}

// run 讀設定、接 signal、啟動服務。signal 只在這裡處理，Service 只看到一個會被取消的 context。
func run(log *slog.Logger) error {
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	// 第一個訊號開始關機；之後恢復預設行為，關機卡住時再按一次就能直接結束。
	context.AfterFunc(ctx, stop)

	svc := &reminder.Service{Config: cfg, Logger: log}
	return svc.Run(ctx)
}

// ANCHOR_END: main
