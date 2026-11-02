// Package reminder 把提醒服務組起來：開資料庫、套用 migration、建立推播客戶端與掃描送出的迴圈，
// 並決定收到結束訊號之後各部分關閉的順序。
//
// 這是唯一知道所有零件的地方；cmd/reminder 的 main 只負責讀環境變數、處理 signal，再呼叫 Service.Run。
// 各零件只依賴它們用到的東西：
//
//	config      不依賴任何人
//	store       資料層，依賴 sqlite driver
//	push        推播客戶端，只依賴標準庫
//	dispatch    掃描與送出，依賴 store 與 push（Sender 是定義在使用端的介面）
//	reminder    依賴以上全部
package reminder

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter16/reminder/config"
	"github.com/holmeslin/learning-go-to-senior/examples/chapter16/reminder/dispatch"
	"github.com/holmeslin/learning-go-to-senior/examples/chapter16/reminder/push"
	"github.com/holmeslin/learning-go-to-senior/examples/chapter16/reminder/store"
)

// databaseMaxConns 是資料庫連線池的上限。送出的並行上限（預設 20）不必等於它：
// 每一則只在寫紀錄與記結果時各占一條連線一小段時間，呼叫供應商的期間不占連線；
// sqlite 同一時間又只有一個寫入者，連線再多也只是排隊。
const databaseMaxConns = 4

// Service 是提醒服務。
type Service struct {
	// Config 是 config.Load 驗證過的設定。
	Config config.Config
	// Logger 記錄服務的事件；nil 時用 slog.Default()。日誌不記裝置 token。
	Logger *slog.Logger
	// Transport 是呼叫推播供應商用的傳輸；nil 時用 http.DefaultTransport。
	// 端對端測試用假的 RoundTripper 取代網路：synctest 的 bubble 裡，socket I/O 不算 durably blocked，
	// 假時間不會前進；這個欄位就是為了讓測試能把網路換掉。
	Transport http.RoundTripper
}

// ANCHOR: run

// Run 啟動服務，直到 ctx 結束，再依序關機；正常關機回傳 nil。
//
// 啟動時先套用自己的 migration，並確認報名服務的唯讀檢視讀得到、欄位齊全，否則拒絕啟動。
// 送出的開關關閉時，服務照常啟動，但不掃描、不寫任何紀錄、不呼叫供應商。
// 上線的第一步就是以這個狀態部署，此時報名服務裡的舊 worker 還在運作，新服務不能碰它的提醒。
//
// ctx 結束（收到 SIGTERM）之後的順序，期限從這一刻起算，全部包含在 Config.ShutdownTimeout 之內：
//
//  1. 停止新的掃描：掃描迴圈不再拿新的提醒。
//  2. 等進行中的送出做完：每一則最久是一個請求逾時，期限大於它加上餘裕，所以正常會在期限內結束；
//     期限到了仍未結束，回傳錯誤並記錄。
//  3. 關資料庫：一定在送出都結束之後，進行中的送出才寫得進結果。
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

	scanDone := make(chan struct{})
	if cfg.SendEnabled {
		d, err := dispatch.New(st, push.NewClient(cfg.ProviderURL, cfg.RequestTimeout, s.Transport), dispatch.Options{
			Interval:      cfg.ScanInterval,
			Concurrency:   cfg.SendConcurrency,
			RatePerSecond: cfg.RateLimit,
			QueryTimeout:  cfg.QueryTimeout,
			Logger:        log,
		})
		if err != nil {
			return errors.Join(fmt.Errorf("create dispatcher: %w", err), db.Close())
		}
		go func() {
			defer close(scanDone)
			d.Run(ctx)
		}()
		log.Info("reminder service started", "send_enabled", true)
	} else {
		close(scanDone)
		log.Info("reminder service started", "send_enabled", false)
	}

	<-ctx.Done() // 收到結束訊號；掃描迴圈已經不再拿新的提醒
	log.Info("shutting down", "timeout", cfg.ShutdownTimeout.String())
	return s.shutdown(ctx, log, scanDone, db)
}

// ANCHOR_END: run

// ANCHOR: shutdown

// shutdown 依序關閉各部分。ctx 已經結束，期限要從沒有被取消的 context 另外算。
func (s *Service) shutdown(ctx context.Context, log *slog.Logger, scanDone <-chan struct{}, db *sql.DB) error {
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
	if err := db.Close(); err != nil {
		errs = append(errs, fmt.Errorf("close database: %w", err))
	}
	return errors.Join(errs...)
}

// ANCHOR_END: shutdown
