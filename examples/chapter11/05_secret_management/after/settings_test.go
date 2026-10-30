package settings_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	settings "github.com/holmeslin/learning-go-to-senior/examples/chapter11/05_secret_management/after"
)

// ANCHOR: load

// 假的秘密。用 t.Setenv 的測試不能呼叫 t.Parallel。
const (
	apiKey     = "example-not-a-real-key"
	dbPassword = "example-not-a-real-password"
)

// load 設好兩個假的秘密並載入設定；空字串的 _FILE 等於沒設，不受開發機環境影響。
func load(t *testing.T) settings.Config {
	t.Helper()
	t.Setenv("REPORT_API_KEY", apiKey)
	t.Setenv("REPORT_API_KEY_FILE", "")
	t.Setenv("REPORT_DB_PASSWORD", dbPassword)
	t.Setenv("REPORT_DB_PASSWORD_FILE", "")
	cfg, err := settings.Load()
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func assertRedacted(t *testing.T, name, out string) {
	t.Helper()
	for _, secret := range []string{apiKey, dbPassword} {
		if strings.Contains(out, secret) {
			t.Errorf("%s leaks %q: %s", name, secret, out)
		}
	}
	if !strings.Contains(out, "[REDACTED]") {
		t.Errorf("%s has no redaction marker: %s", name, out)
	}
}

// ANCHOR_END: load

// ANCHOR: fmt

func TestFmtNeverPrintsSecrets(t *testing.T) {
	cfg := load(t)
	values := map[string]any{
		"secret": cfg.APIKey, "config": cfg, "pointer": &cfg, "dsn": cfg.DSN(),
		"slice": []settings.Secret{cfg.APIKey}, "map": map[string]settings.Secret{"k": cfg.APIKey},
	}
	for name, v := range values {
		for _, verb := range []string{"%v", "%+v", "%#v", "%s", "%q"} {
			assertRedacted(t, name+" "+verb, fmt.Sprintf(verb, v))
		}
	}
}

// ANCHOR_END: fmt

func TestSlogAndJSONNeverPrintSecrets(t *testing.T) {
	cfg := load(t)

	for name, newHandler := range map[string]func(*bytes.Buffer) slog.Handler{
		"text": func(b *bytes.Buffer) slog.Handler { return slog.NewTextHandler(b, nil) },
		"json": func(b *bytes.Buffer) slog.Handler { return slog.NewJSONHandler(b, nil) },
	} {
		var buf bytes.Buffer
		slog.New(newHandler(&buf)).Info("start", "api_key", cfg.APIKey, "config", cfg, "dsn", cfg.DSN())
		assertRedacted(t, "slog "+name, buf.String())
	}

	for name, v := range map[string]any{"config": cfg, "secret": cfg.APIKey, "map": map[string]any{"k": cfg.DSN()}} {
		out, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		assertRedacted(t, "json "+name, string(out))
		out, err = json.MarshalIndent(v, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		assertRedacted(t, "indented json "+name, string(out))
	}
}

func TestLogValueIsResolvedBySlog(t *testing.T) {
	cfg := load(t)
	// 沒有 LogValue 時 Kind 是 KindAny，要看 handler 怎麼格式化；有的話 slog 先換成已遮蔽的字串。
	if got := slog.AnyValue(cfg.APIKey).Resolve(); got.Kind() != slog.KindString || got.String() != "[REDACTED]" {
		t.Errorf("resolved = %v (%v), want the redacted string", got, got.Kind())
	}
}

func TestRevealIsTheOnlyWayOut(t *testing.T) {
	cfg := load(t)
	if got := cfg.APIKey.Reveal(); got != apiKey {
		t.Errorf("Reveal() = %q, want %q", got, apiKey)
	}
}

// ANCHOR: gaps

// 型別擋不住的兩種情況，鎖住取捨段的說法：錯的動詞，以及小寫欄位（fmt 看不到方法）。
func TestKnownGaps(t *testing.T) {
	cfg := load(t)
	wrongVerb := "%d" // go vet 抓得到寫死的 %d，所以放進變數
	//nolint:staticcheck // SA5009：staticcheck 連變數裡的 %d 都抓得到，這裡刻意用錯的動詞
	if out := fmt.Sprintf(wrongVerb, cfg.APIKey); !strings.Contains(out, apiKey) {
		t.Errorf("%%d of a Secret = %q, expected the documented gap", out)
	}
	holder := struct{ key settings.Secret }{cfg.APIKey}
	if out := fmt.Sprintf("%v", holder); !strings.Contains(out, apiKey) {
		t.Errorf("unexported field = %q, expected the documented gap", out)
	}
}

// ANCHOR_END: gaps

func TestLoadFromEnvironment(t *testing.T) {
	cfg := load(t)
	if cfg.APIKey.Reveal() != apiKey || cfg.DSN().Reveal() != "postgres://report:"+dbPassword+"@db.internal:5432/reports" {
		t.Errorf("loaded %v, want the values from the environment", cfg)
	}
}

func TestLoadFromFileTrimsNewline(t *testing.T) {
	load(t)
	path := filepath.Join(t.TempDir(), "api_key")
	if err := os.WriteFile(path, []byte(apiKey+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("REPORT_API_KEY", "")
	t.Setenv("REPORT_API_KEY_FILE", path)
	cfg, err := settings.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.APIKey.Reveal(); got != apiKey {
		t.Errorf("Reveal() = %q, want %q", got, apiKey)
	}
}

// ANCHOR: failfast

func TestLoadFailsWhenSecretIsMissing(t *testing.T) {
	tests := []struct {
		name, key, password string
		wantNames           []string
		leaked              string
	}{
		{"neither", "", "", []string{"REPORT_API_KEY", "REPORT_DB_PASSWORD"}, ""},
		{"no password", apiKey, "", []string{"REPORT_DB_PASSWORD"}, apiKey},
		{"no key", "", dbPassword, []string{"REPORT_API_KEY"}, dbPassword},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			load(t)
			t.Setenv("REPORT_API_KEY", tt.key)
			t.Setenv("REPORT_DB_PASSWORD", tt.password)
			_, err := settings.Load()
			if err == nil {
				t.Fatal("Load succeeded without a secret")
			}
			for _, name := range tt.wantNames {
				if !strings.Contains(err.Error(), name) {
					t.Errorf("error %q does not name %s", err, name)
				}
			}
			if tt.leaked != "" && strings.Contains(err.Error(), tt.leaked) {
				t.Errorf("error %q contains a secret that was set", err)
			}
		})
	}
}

// ANCHOR_END: failfast

func TestLoadFailsOnUnreadableOrEmptyFile(t *testing.T) {
	load(t)
	t.Setenv("REPORT_API_KEY_FILE", filepath.Join(t.TempDir(), "missing"))
	if _, err := settings.Load(); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("missing file: err = %v, want fs.ErrNotExist", err)
	}

	empty := filepath.Join(t.TempDir(), "empty")
	if err := os.WriteFile(empty, []byte("\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("REPORT_API_KEY_FILE", empty)
	if _, err := settings.Load(); err == nil {
		t.Error("an empty secret file was accepted")
	}
}

// ANCHOR: connect

func TestConnectErrorOmitsDSN(t *testing.T) {
	cfg := load(t)
	errRefused := errors.New("connection refused")

	err := settings.Connect(cfg, func(string) error { return errRefused })
	if !errors.Is(err, errRefused) {
		t.Fatalf("err = %v, want it to wrap the cause", err)
	}
	if msg := err.Error(); !strings.Contains(msg, "db.internal:5432") || strings.Contains(msg, dbPassword) {
		t.Errorf("error = %q, want the host and no password", msg)
	}

	var dsn string
	if err := settings.Connect(cfg, func(s string) error { dsn = s; return nil }); err != nil || !strings.Contains(dsn, dbPassword) {
		t.Errorf("open got dsn %q, err %v; want the real DSN to reach the driver", dsn, err)
	}
}

// ANCHOR_END: connect
