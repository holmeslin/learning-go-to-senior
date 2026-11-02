// Package reminder 把提醒服務組起來：開資料庫、套用 migration、建立指標與追蹤、推播客戶端、掃描送出的迴圈、
// 內部 HTTP API 與診斷端點，並決定收到結束訊號之後各部分關閉的順序。
//
// 這是唯一知道所有零件的地方；cmd/reminder 的 main 只負責讀環境變數、處理 signal，再呼叫 Service.Run。
// 各零件只依賴它們用到的東西：
//
//	config      不依賴任何人
//	store       資料層，依賴 sqlite driver
//	push        推播客戶端，只依賴標準庫
//	telemetry   指標與追蹤（OpenTelemetry SDK）
//	dispatch    掃描與送出，依賴 store 與 push（Sender 是定義在使用端的介面）
//	api         內部 HTTP API，依賴 store 的型別（Counter 是定義在使用端的介面）
//	buildinfo   版本資訊，只依賴標準庫
//	diag        診斷端點，依賴 buildinfo 的型別
//	reminder    依賴以上全部
package reminder

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter16/reminder/internal/api"
	"github.com/holmeslin/learning-go-to-senior/examples/chapter16/reminder/internal/buildinfo"
	"github.com/holmeslin/learning-go-to-senior/examples/chapter16/reminder/internal/config"
	"github.com/holmeslin/learning-go-to-senior/examples/chapter16/reminder/internal/diag"
	"github.com/holmeslin/learning-go-to-senior/examples/chapter16/reminder/internal/dispatch"
	"github.com/holmeslin/learning-go-to-senior/examples/chapter16/reminder/internal/push"
	"github.com/holmeslin/learning-go-to-senior/examples/chapter16/reminder/internal/store"
	"github.com/holmeslin/learning-go-to-senior/examples/chapter16/reminder/internal/telemetry"
)

// databaseMaxConns 是資料庫連線池的上限。送出的並行上限（預設 50）不必等於它：
// 每一則只在寫紀錄與記結果時各占一條連線一小段時間，呼叫供應商的期間不占連線；
// sqlite 同一時間又只有一個寫入者，連線再多也只是排隊。
const databaseMaxConns = 4

// ANCHOR: service

// Service 是提醒服務。
type Service struct {
	// Config 是 config.Load 驗證過的設定。Config.APIAddr 或 Config.DiagAddr 是空字串時，
	// Service 不啟動那個 HTTP 伺服器：Load 不會產生空字串，這是留給測試的接縫（synctest 的 bubble 裡不能有真的 socket）。
	Config config.Config
	// Build 是這個執行檔的版本資訊（見 buildinfo.From）：啟動日誌與診斷端點的 /version 用它。零值代表沒有。
	Build buildinfo.Info
	// Logger 記錄服務的事件；nil 時用 slog.Default()。日誌不記裝置 token。
	Logger *slog.Logger
	// Transport 是呼叫推播供應商用的傳輸；nil 時用 http.DefaultTransport。
	// 端對端測試用假的 RoundTripper 取代網路：synctest 的 bubble 裡，socket I/O 不算 durably blocked，
	// 假時間不會前進；這個欄位就是為了讓測試能把網路換掉。
	Transport http.RoundTripper
	// SpanExporter 是追蹤送去的地方；nil 代表不送出（OTLP 不在 go.mod 裡，見 telemetry.New 的說明）。
	// 測試用記憶體的 exporter 檢查關機時送出了什麼；上線時在 main 接真正的 exporter。
	SpanExporter sdktrace.SpanExporter
}

// ANCHOR_END: service

// ANCHOR: run

// Run 啟動服務，直到 ctx 結束，再依序關機；正常關機回傳 nil。
//
// 啟動時先套用自己的 migration，並確認報名服務的唯讀檢視讀得到、欄位齊全，否則拒絕啟動。
// 送出的開關關閉時，服務照常啟動，但不掃描、不寫任何紀錄、不呼叫供應商。
// 上線的第一步就是以這個狀態部署，此時報名服務裡的舊 worker 還在運作，新服務不能碰它的提醒。
//
// ctx 結束（收到 SIGTERM）之後的順序，期限從這一刻起算，全部包含在 Config.ShutdownTimeout 之內：
//
//  1. readiness 先變成未就緒：流量與編排系統先知道這個實例要走了。
//  2. 停止新的掃描：掃描迴圈不再拿新的提醒。
//  3. 等進行中的送出做完：每一則最久是寫紀錄的查詢、請求、記結果的查詢各自逾時
//     （QueryTimeout 兩次加 RequestTimeout 一次，預設共 15 秒）。
//  4. 關 HTTP 伺服器（內部 API 與診斷端點），等進行中的請求，最久再一個 QueryTimeout；再送出指標與追蹤。
//  5. 關資料庫：一定在最後，進行中的送出才寫得進結果，收集指標的查詢也還查得到。
//
// 期限 ShutdownTimeout 要涵蓋第 3、4 步的最壞耗時加第 4 步送出的餘裕：
// config.Load 檢查它至少是 RequestTimeout + 3×QueryTimeout + 5 秒（預設 25 秒，預設的期限是 30 秒）。
// 期限到了第 3 步仍未結束，回傳錯誤並記錄；之後的步驟仍然做，不過用的是已經過期的 context。
//
// shortcut: 轉成未就緒之後立刻停掃描，沒有再等一段時間讓負載平衡發現；這個服務是單一實例、
// 內部 API 前面沒有負載平衡，不需要。有多個副本時，在第 1、2 步之間加上等待。
func (s *Service) Run(ctx context.Context) error {
	log := s.Logger
	if log == nil {
		log = slog.Default()
	}
	cfg := s.Config

	db, err := store.OpenDB(cfg.DatabasePath, databaseMaxConns)
	if err != nil {
		return err
	}
	st := store.New(db)
	if err := store.Migrate(ctx, db); err != nil {
		return errors.Join(fmt.Errorf("migrate: %w", err), db.Close())
	}
	if err := st.CheckCandidates(ctx); err != nil {
		return errors.Join(err, db.Close())
	}
	tel, err := telemetry.New(s.SpanExporter)
	if err != nil {
		return errors.Join(err, db.Close())
	}
	// 啟動到一半失敗：把已經開起來的東西關掉，連同原因一起回傳。
	abort := func(cause error, started ...*server) error {
		errs := []error{cause, tel.Shutdown(context.WithoutCancel(ctx)), db.Close()}
		for _, srv := range started {
			errs = append(errs, srv.close(context.WithoutCancel(ctx)))
		}
		return errors.Join(errs...)
	}

	// 先做會失敗、又不占資源的事（建立 Dispatcher），再開 listener。
	var d *dispatch.Dispatcher
	if cfg.SendEnabled {
		d, err = dispatch.New(st, push.NewClient(cfg.ProviderURL, cfg.RequestTimeout, s.Transport), dispatch.Options{
			Interval:      cfg.ScanInterval,
			Concurrency:   cfg.SendConcurrency,
			RatePerSecond: cfg.RateLimit,
			QueryTimeout:  cfg.QueryTimeout,
			Logger:        log,
			Meter:         tel.Meter,
			Tracer:        tel.Tracer,
		})
		if err != nil {
			return abort(fmt.Errorf("create dispatcher: %w", err))
		}
	}
	ready := &diag.Readiness{}
	apiSrv, err := startServer(log, "api", cfg.APIAddr, api.NewHandler(st, cfg.QueryTimeout, log))
	if err != nil {
		return abort(err)
	}
	diagSrv, err := startServer(log, "diagnostic", cfg.DiagAddr, diag.NewHandler(ready, tel, s.Build))
	if err != nil {
		return abort(err, apiSrv)
	}

	// 掃描用自己的 context：關機時要先把 readiness 設成未就緒，才輪到停止新的掃描。
	scanCtx, stopScanning := context.WithCancel(context.WithoutCancel(ctx))
	defer stopScanning()
	scanDone := make(chan struct{})
	if d != nil {
		go func() {
			defer close(scanDone)
			d.Run(scanCtx)
		}()
	} else {
		close(scanDone)
	}
	ready.Set(true)
	log.Info("reminder service started", "send_enabled", cfg.SendEnabled, "version", s.Build.String())

	<-ctx.Done() // 收到結束訊號
	ready.Set(false)
	stopScanning()
	log.Info("shutting down", "timeout", cfg.ShutdownTimeout.String())
	return s.shutdown(ctx, log, scanDone, []*server{apiSrv, diagSrv}, tel, db)
}

// ANCHOR_END: run

// ANCHOR: shutdown

// shutdown 依序做完 Run 的第 3 到第 5 步。ctx 已經結束，期限要從沒有被取消的 context 另外算。
func (s *Service) shutdown(ctx context.Context, log *slog.Logger, scanDone <-chan struct{}, servers []*server, tel *telemetry.Telemetry, db *sql.DB) error {
	grace, cancel := context.WithTimeout(context.WithoutCancel(ctx), s.Config.ShutdownTimeout)
	defer cancel()

	var errs []error
	select {
	case <-scanDone:
	case <-grace.Done():
		err := fmt.Errorf("in-flight sends did not finish: %w", grace.Err())
		log.Error("shutdown deadline passed", "error", err)
		errs = append(errs, err)
	}
	// 兩個伺服器同時收尾：等進行中的請求，最久各是一個 QueryTimeout，同時等就只要一個。
	// pprof 的 profile 可以跑 30 秒，超過期限的請求會被強制切斷，診斷用的請求可以這樣處理。
	closeErrs := make([]error, len(servers))
	var wg sync.WaitGroup
	for i, srv := range servers {
		wg.Go(func() { closeErrs[i] = srv.close(grace) })
	}
	wg.Wait()
	errs = append(errs, closeErrs...)
	if err := tel.Shutdown(grace); err != nil {
		errs = append(errs, fmt.Errorf("shutdown telemetry: %w", err))
	}
	if err := db.Close(); err != nil {
		errs = append(errs, fmt.Errorf("close database: %w", err))
	}
	return errors.Join(errs...)
}

// ANCHOR_END: shutdown

// server 是一個在背景服務的 HTTP 伺服器。
type server struct {
	name string
	srv  *http.Server
}

// startServer 在 addr 上開始服務 h；addr 是空字串代表不啟動，回傳 nil。
// 先同步地 Listen：位址被占用時啟動就失敗，不是等到之後才發現沒有人在聽。
func startServer(log *slog.Logger, name, addr string, h http.Handler) (*server, error) {
	if addr == "" {
		return nil, nil
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen for %s server: %w", name, err)
	}
	// 診斷端點不能設 WriteTimeout（pprof 的 profile 預設要跑 30 秒），所以兩個伺服器都只限制讀標頭的時間。
	srv := &http.Server{Handler: h, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		// shortcut: Serve 失敗（關閉以外的原因）只記錄，服務繼續跑；listener 已經開起來，這很少發生。
		// 要讓它變成致命的，把錯誤送回 Run，由 Run 開始關機。
		if err := srv.Serve(ln); !errors.Is(err, http.ErrServerClosed) {
			log.Error(name+" server stopped", "error", err)
		}
	}()
	log.Info(name+" server listening", "addr", ln.Addr().String())
	return &server{name: name, srv: srv}, nil
}

// close 等進行中的請求結束再關；ctx 的期限到了就強制關掉。對 nil（沒有啟動的伺服器）什麼也不做。
func (s *server) close(ctx context.Context) error {
	if s == nil {
		return nil
	}
	err := s.srv.Shutdown(ctx)
	if err == nil {
		return nil
	}
	return fmt.Errorf("shutdown %s server: %w", s.name, errors.Join(err, s.srv.Close()))
}
