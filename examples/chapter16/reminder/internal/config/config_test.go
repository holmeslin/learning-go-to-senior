package config_test

import (
	"maps"
	"strings"
	"testing"
	"time"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter16/reminder/internal/config"
)

// ANCHOR: env

// env 把 map 當成環境變數：測試不碰行程的環境，也不需要 t.Setenv。
func env(vars map[string]string) func(string) string {
	return func(key string) string { return vars[key] }
}

// ANCHOR_END: env

func TestLoadDefaults(t *testing.T) {
	t.Parallel()
	got, err := config.Load(env(map[string]string{"REMINDER_PROVIDER_URL": "https://push.example.com"}))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	want := config.Config{
		SendEnabled:     false, // 送出的開關預設關閉
		ScanInterval:    5 * time.Minute,
		SendConcurrency: 50,
		RequestTimeout:  5 * time.Second,
		RateLimit:       50,
		QueryTimeout:    5 * time.Second,
		ShutdownTimeout: 30 * time.Second,
		DatabasePath:    "reminder.db",
		ProviderURL:     "https://push.example.com",
		APIAddr:         ":8080",
		DiagAddr:        "127.0.0.1:6060",
	}
	if got != want {
		t.Errorf("Load() = %+v, want %+v", got, want)
	}
}

func TestLoadOverrides(t *testing.T) {
	t.Parallel()
	got, err := config.Load(env(map[string]string{
		"REMINDER_SEND_ENABLED":     "true",
		"REMINDER_SCAN_INTERVAL":    "1m",
		"REMINDER_SEND_CONCURRENCY": "8",
		"REMINDER_REQUEST_TIMEOUT":  "2s",
		"REMINDER_RATE_LIMIT":       "12.5",
		"REMINDER_QUERY_TIMEOUT":    "1s",
		"REMINDER_SHUTDOWN_TIMEOUT": "10s",
		"REMINDER_DB_PATH":          "/var/lib/reminder/reminder.db",
		"REMINDER_PROVIDER_URL":     "http://127.0.0.1:9000",
		"REMINDER_API_ADDR":         "0.0.0.0:9090",
		"REMINDER_DIAG_ADDR":        "[::1]:9091",
	}))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	want := config.Config{
		SendEnabled:     true,
		ScanInterval:    time.Minute,
		SendConcurrency: 8,
		RequestTimeout:  2 * time.Second,
		RateLimit:       12.5,
		QueryTimeout:    time.Second,
		ShutdownTimeout: 10 * time.Second,
		DatabasePath:    "/var/lib/reminder/reminder.db",
		ProviderURL:     "http://127.0.0.1:9000",
		APIAddr:         "0.0.0.0:9090",
		DiagAddr:        "[::1]:9091",
	}
	if got != want {
		t.Errorf("Load() = %+v, want %+v", got, want)
	}
}

// ANCHOR: invalid

func TestLoadRejectsInvalidValues(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		key  string
		val  string
	}{
		{"switch is not a bool", "REMINDER_SEND_ENABLED", "yes please"},
		{"scan interval is not a duration", "REMINDER_SCAN_INTERVAL", "often"},
		{"scan interval is zero", "REMINDER_SCAN_INTERVAL", "0s"},
		{"concurrency is not a number", "REMINDER_SEND_CONCURRENCY", "many"},
		{"concurrency is zero", "REMINDER_SEND_CONCURRENCY", "0"},
		{"request timeout is negative", "REMINDER_REQUEST_TIMEOUT", "-1s"},
		{"rate limit is not a number", "REMINDER_RATE_LIMIT", "fast"},
		{"rate limit is zero", "REMINDER_RATE_LIMIT", "0"},
		{"query timeout is zero", "REMINDER_QUERY_TIMEOUT", "0s"},
		{"shutdown timeout is shorter than the worst-case send", "REMINDER_SHUTDOWN_TIMEOUT", "5s"},
		{"provider url has no host", "REMINDER_PROVIDER_URL", "https://"},
		{"provider url is not http", "REMINDER_PROVIDER_URL", "ftp://push.example.com"},
		{"api address has no port", "REMINDER_API_ADDR", "localhost"},
		{"api address port is out of range", "REMINDER_API_ADDR", ":70000"},
		{"diagnostics address is not loopback", "REMINDER_DIAG_ADDR", "0.0.0.0:6060"},
		{"diagnostics address has no host", "REMINDER_DIAG_ADDR", ":6060"},
		{"diagnostics address is a public host", "REMINDER_DIAG_ADDR", "push.example.com:6060"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			vars := map[string]string{"REMINDER_PROVIDER_URL": "https://push.example.com", tt.key: tt.val}
			_, err := config.Load(env(vars))
			if err == nil {
				t.Fatalf("Load() with %s=%q error = nil, want an error", tt.key, tt.val)
			}
			// 錯誤指出是哪個變數出的問題。
			if !strings.Contains(err.Error(), tt.key) {
				t.Errorf("Load() error = %q, want it to name %s", err, tt.key)
			}
		})
	}
}

// ANCHOR_END: invalid

// ANCHOR: shutdownbudget

// 關機期限要涵蓋最壞的一則送出：寫紀錄的查詢、請求、記結果的查詢，加上 HTTP 伺服器收尾的一個查詢，
// 再加 5 秒餘裕。預設的請求與查詢逾時都是 5 秒，所以下限是 5 + 3×5 + 5 = 25 秒。
func TestLoadChecksTheShutdownTimeoutAgainstTheWorstCaseSend(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		vars map[string]string
		ok   bool
	}{
		{"just below the minimum", map[string]string{"REMINDER_SHUTDOWN_TIMEOUT": "24s"}, false},
		{"exactly the minimum", map[string]string{"REMINDER_SHUTDOWN_TIMEOUT": "25s"}, true},
		{"the default", nil, true},
		// 查詢逾時調大，下限跟著變：5 + 3×10 + 5 = 40 秒，預設的 30 秒不夠了。
		{"a longer query timeout needs a longer shutdown timeout", map[string]string{"REMINDER_QUERY_TIMEOUT": "10s"}, false},
		{"and then it fits", map[string]string{"REMINDER_QUERY_TIMEOUT": "10s", "REMINDER_SHUTDOWN_TIMEOUT": "40s"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			vars := map[string]string{"REMINDER_PROVIDER_URL": "https://push.example.com"}
			maps.Copy(vars, tt.vars)
			_, err := config.Load(env(vars))
			if tt.ok {
				if err != nil {
					t.Errorf("Load() error = %v, want none", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), "REMINDER_SHUTDOWN_TIMEOUT") {
				t.Errorf("Load() error = %v, want it to name REMINDER_SHUTDOWN_TIMEOUT", err)
			}
		})
	}
}

// 錯誤訊息直接說出下限與算式，改設定的人不必回頭查程式。
func TestLoadShutdownTimeoutErrorStatesTheMinimum(t *testing.T) {
	t.Parallel()
	_, err := config.Load(env(map[string]string{"REMINDER_PROVIDER_URL": "https://push.example.com", "REMINDER_SHUTDOWN_TIMEOUT": "20s"}))
	want := "REMINDER_SHUTDOWN_TIMEOUT: must be at least 25s (REMINDER_REQUEST_TIMEOUT + 3 x REMINDER_QUERY_TIMEOUT + 5s)"
	if err == nil || err.Error() != want {
		t.Errorf("Load() error = %v, want %q", err, want)
	}
}

// ANCHOR_END: shutdownbudget

func TestLoadRequiresProviderURL(t *testing.T) {
	t.Parallel()
	_, err := config.Load(env(nil))
	if err == nil || !strings.Contains(err.Error(), "REMINDER_PROVIDER_URL") {
		t.Errorf("Load() error = %v, want it to name REMINDER_PROVIDER_URL", err)
	}
}

func TestLoadReportsEveryProblemAtOnce(t *testing.T) {
	t.Parallel()
	_, err := config.Load(env(map[string]string{
		"REMINDER_SEND_CONCURRENCY": "0",
		"REMINDER_RATE_LIMIT":       "fast",
	}))
	if err == nil {
		t.Fatal("Load() error = nil, want errors")
	}
	for _, key := range []string{"REMINDER_SEND_CONCURRENCY", "REMINDER_RATE_LIMIT", "REMINDER_PROVIDER_URL"} {
		if !strings.Contains(err.Error(), key) {
			t.Errorf("Load() error = %q, want it to name %s", err, key)
		}
	}
}

func TestLoadErrorDoesNotEchoTheProviderURL(t *testing.T) {
	t.Parallel()
	// 網址可能夾帶帳號密碼，錯誤訊息不能把它帶出去。
	_, err := config.Load(env(map[string]string{"REMINDER_PROVIDER_URL": "ftp://user:hunter2@push.example.com"}))
	if err == nil {
		t.Fatal("Load() error = nil, want an error")
	}
	if strings.Contains(err.Error(), "hunter2") {
		t.Errorf("Load() error = %q, must not contain the URL's password", err)
	}
}

func TestLoadTreatsEmptyValueAsUnset(t *testing.T) {
	t.Parallel()
	got, err := config.Load(env(map[string]string{"REMINDER_PROVIDER_URL": "https://push.example.com", "REMINDER_SCAN_INTERVAL": ""}))
	if err != nil || got.ScanInterval != 5*time.Minute {
		t.Errorf("Load() = %v, %v, want the default scan interval", got.ScanInterval, err)
	}
}
