// Command reminder 是提醒服務的行程入口：讀環境變數、處理 signal、組裝，其他的事都交給 reminder.Service。
//
// 執行時至少要設 REMINDER_PROVIDER_URL；送出的開關（REMINDER_SEND_ENABLED）預設關閉。
// 版本由建置時的 -ldflags 注入（見 version）；啟動日誌與診斷端點的 /version 看得到。
// 全部的設定名稱與預設值見 config.Config。內部 API 預設在 :8080，診斷端點（健康檢查、readiness、
// /debug/pprof、/debug/metrics）預設在 127.0.0.1:6060，只聽 loopback。
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter16/reminder"
	"github.com/holmeslin/learning-go-to-senior/examples/chapter16/reminder/internal/buildinfo"
	"github.com/holmeslin/learning-go-to-senior/examples/chapter16/reminder/internal/config"
)

// ANCHOR: version

// version 是發布流程決定的版本號，建置時用 -ldflags "-X main.version=..." 注入；
// 沒注入就是空字串，這時版本取自 Go 記在執行檔裡的建置資訊（debug.ReadBuildInfo）。
var version string

// ANCHOR_END: version

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

	bi, _ := debug.ReadBuildInfo() // 讀不到時是 nil，buildinfo.From 會處理
	svc := &reminder.Service{Config: cfg, Logger: log, Build: buildinfo.From(bi, version)}
	return svc.Run(ctx)
}

// ANCHOR_END: main
