// Package config 讀取提醒服務的設定：全部來自環境變數，啟動時驗證，有問題就不啟動。
//
// 設定的做法見第 12 章〈設定與環境變數〉；這裡的 Load 接收一個查詢環境變數的函式，
// 測試不必改動行程的環境。拿到 Config 的程式不必再檢查欄位，也不該再讀環境變數。
package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"
)

// ANCHOR: defaults

// 預設值。送出的開關預設關閉：新服務先以關閉的狀態部署，確認沒問題才打開。
const (
	defaultScanInterval = 5 * time.Minute
	// 並行上限取和限流同一個數字：供應商每則在 1 秒內回應時，瓶頸是限流，不是這個上限；
	// 每則都拖滿 RequestTimeout 的最壞情況，吞吐量仍是每秒 10 則，兩萬人的第一輪約 33 分鐘，在 1 小時之內。
	defaultSendConcurrency = 50
	defaultRequestTimeout  = 5 * time.Second
	defaultRateLimit       = 50.0 // 推播供應商的限制是每秒 50 次
	defaultQueryTimeout    = 5 * time.Second
	defaultShutdownTimeout = 30 * time.Second
	defaultDatabasePath    = "reminder.db"
	defaultAPIAddr         = ":8080"
	defaultDiagAddr        = "127.0.0.1:6060"

	// shutdownMargin 是關機期限在最壞耗時之外留的餘裕，給送出指標與追蹤、關資料庫。
	// 最壞耗時是 RequestTimeout 加三個 QueryTimeout：進行中的一則最久要等寫紀錄的查詢、請求、記結果的查詢，
	// 各自逾時（QueryTimeout 兩次、RequestTimeout 一次）；再來 HTTP 伺服器收尾，等進行中的 API 請求，
	// 請求裡只有一個查詢（QueryTimeout 一次）。所以 ShutdownTimeout 至少是
	// RequestTimeout + 3×QueryTimeout + shutdownMargin，預設是 5s + 3×5s + 5s = 25s，預設的期限是 30 秒。
	shutdownMargin = 5 * time.Second
)

// ANCHOR_END: defaults

// ANCHOR: config

// Config 是提醒服務的設定。括號裡是環境變數的名稱與預設值。
type Config struct {
	// SendEnabled 是送出的開關（REMINDER_SEND_ENABLED，預設 false）。關閉時不掃描、不寫任何紀錄、不送出。
	SendEnabled bool
	// ScanInterval 是掃描的間隔（REMINDER_SCAN_INTERVAL，預設 5m），必須大於零。
	ScanInterval time.Duration
	// SendConcurrency 是同時進行中的送出上限（REMINDER_SEND_CONCURRENCY，預設 50），至少 1。
	// 它和供應商每秒 50 次的限流是兩件事：這個擋的是同時有幾個請求在路上，限流擋的是每秒送出幾個。
	SendConcurrency int
	// RequestTimeout 是呼叫供應商時每個請求的逾時（REMINDER_REQUEST_TIMEOUT，預設 5s），必須大於零。
	RequestTimeout time.Duration
	// RateLimit 是每秒最多送出幾則（REMINDER_RATE_LIMIT，預設 50），必須大於零。
	RateLimit float64
	// QueryTimeout 是每個資料庫查詢的逾時（REMINDER_QUERY_TIMEOUT，預設 5s），必須大於零。
	QueryTimeout time.Duration
	// ShutdownTimeout 是收到結束訊號之後，整個關機的期限（REMINDER_SHUTDOWN_TIMEOUT，預設 30s）。
	// 必須涵蓋最壞的一則送出，至少是 RequestTimeout + 3×QueryTimeout + 5 秒，見 shutdownMargin。
	ShutdownTimeout time.Duration
	// DatabasePath 是 sqlite 資料庫的路徑（REMINDER_DB_PATH，預設 reminder.db）。
	DatabasePath string
	// ProviderURL 是推播供應商的網址（REMINDER_PROVIDER_URL，必填），必須是帶主機的 http 或 https 網址。
	ProviderURL string
	// APIAddr 是內部 HTTP API 監聽的位址（REMINDER_API_ADDR，預設 :8080）。
	APIAddr string
	// DiagAddr 是診斷端點監聽的位址（REMINDER_DIAG_ADDR，預設 127.0.0.1:6060），主機必須是 loopback。
	DiagAddr string
}

// ANCHOR_END: config

// ANCHOR: load

// Load 從 env 查詢環境變數，轉換、驗證之後回傳 Config。環境變數是空字串視為沒設。
// 每個有問題的變數都會回報，用 errors.Join 一次回傳；錯誤訊息帶變數名稱，不帶網址的內容（可能夾帶帳號密碼）。
func Load(env func(key string) string) (Config, error) {
	l := loader{env: env}
	cfg := Config{
		SendEnabled:     l.boolean("REMINDER_SEND_ENABLED", false),
		ScanInterval:    l.duration("REMINDER_SCAN_INTERVAL", defaultScanInterval),
		SendConcurrency: l.integer("REMINDER_SEND_CONCURRENCY", defaultSendConcurrency),
		RequestTimeout:  l.duration("REMINDER_REQUEST_TIMEOUT", defaultRequestTimeout),
		RateLimit:       l.float("REMINDER_RATE_LIMIT", defaultRateLimit),
		QueryTimeout:    l.duration("REMINDER_QUERY_TIMEOUT", defaultQueryTimeout),
		ShutdownTimeout: l.duration("REMINDER_SHUTDOWN_TIMEOUT", defaultShutdownTimeout),
		DatabasePath:    l.str("REMINDER_DB_PATH", defaultDatabasePath),
		ProviderURL:     l.str("REMINDER_PROVIDER_URL", ""),
		APIAddr:         l.str("REMINDER_API_ADDR", defaultAPIAddr),
		DiagAddr:        l.str("REMINDER_DIAG_ADDR", defaultDiagAddr),
	}
	l.check("REMINDER_SEND_CONCURRENCY", cfg.SendConcurrency >= 1, "must be at least 1")
	minShutdown := cfg.RequestTimeout + 3*cfg.QueryTimeout + shutdownMargin
	l.check("REMINDER_SHUTDOWN_TIMEOUT", cfg.ShutdownTimeout >= minShutdown, fmt.Sprintf(
		"must be at least %s (REMINDER_REQUEST_TIMEOUT + 3 x REMINDER_QUERY_TIMEOUT + %s)", minShutdown, shutdownMargin))
	l.fail("REMINDER_PROVIDER_URL", checkProviderURL(cfg.ProviderURL))
	l.fail("REMINDER_API_ADDR", checkAddr(cfg.APIAddr, false))
	l.fail("REMINDER_DIAG_ADDR", checkAddr(cfg.DiagAddr, true))
	if err := errors.Join(l.errs...); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// ANCHOR_END: load

// loader 逐一讀取環境變數；轉換失敗的錯誤先收集起來，由 Load 一次回傳。
type loader struct {
	env  func(string) string
	errs []error
}

func (l *loader) fail(key string, err error) {
	if err != nil {
		l.errs = append(l.errs, fmt.Errorf("%s: %w", key, err))
	}
}

func (l *loader) check(key string, ok bool, msg string) {
	if !ok {
		l.fail(key, errors.New(msg))
	}
}

func (l *loader) str(key, def string) string {
	if v := l.env(key); v != "" {
		return v
	}
	return def
}

func (l *loader) boolean(key string, def bool) bool {
	v := l.env(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	l.fail(key, err)
	return b
}

func (l *loader) integer(key string, def int) int {
	v := l.env(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	l.fail(key, err)
	return n
}

func (l *loader) float(key string, def float64) float64 {
	v := l.env(key)
	if v == "" {
		return def
	}
	f, err := strconv.ParseFloat(v, 64)
	l.fail(key, err)
	if err == nil {
		l.check(key, f > 0, "must be positive")
	}
	return f
}

func (l *loader) duration(key string, def time.Duration) time.Duration {
	v := l.env(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	l.fail(key, err)
	if err == nil {
		l.check(key, d > 0, "must be positive")
	}
	return d
}

// checkProviderURL 檢查供應商網址。錯誤不帶網址的內容：網址可能夾帶帳號密碼。
func checkProviderURL(v string) error {
	if v == "" {
		return errors.New("required")
	}
	u, err := url.Parse(v)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return errors.New("want an http or https url with a host")
	}
	return nil
}

// checkAddr 檢查 host:port，port 要是 0 到 65535 的數字。loopbackOnly 時，主機必須是 loopback：
// 診斷端點有 pprof，不能對外。
func checkAddr(v string, loopbackOnly bool) error {
	host, port, err := net.SplitHostPort(v)
	if err != nil {
		return errors.New("want host:port")
	}
	if p, err := strconv.Atoi(port); err != nil || p < 0 || p > 65535 {
		return errors.New("port must be a number between 0 and 65535")
	}
	if loopbackOnly && host != "localhost" && !net.ParseIP(host).IsLoopback() {
		return errors.New("host must be a loopback address such as 127.0.0.1")
	}
	return nil
}
