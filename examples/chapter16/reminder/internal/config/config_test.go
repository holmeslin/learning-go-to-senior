package config_test

import (
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
		SendConcurrency: 20,
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
		{"shutdown timeout leaves no margin over the request timeout", "REMINDER_SHUTDOWN_TIMEOUT", "5s"},
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
