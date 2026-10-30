package wallet_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	wallet "github.com/holmeslin/learning-go-to-senior/examples/chapter11/08_sensitive_data_in_logs/after"
	before "github.com/holmeslin/learning-go-to-senior/examples/chapter11/08_sensitive_data_in_logs/before"
)

// ANCHOR: test

// secrets 是明顯的假值；任何一個出現在日誌輸出裡都算外洩。
var secrets = []string{"example-not-a-real-password", "example-not-a-real-token", "1234567890123456"}

// bind 對 h 送一個綁卡請求，本文與 Authorization 標頭都帶著 secrets。
func bind(h http.Handler) *httptest.ResponseRecorder {
	body := `{"user_id":"u-42","password":"example-not-a-real-password","card_number":"1234567890123456"}`
	req := httptest.NewRequest(http.MethodPost, "/cards", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer example-not-a-real-token")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// parse 把 JSONHandler 寫進 buf 的那一行日誌解析成物件。
func parse(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	var rec map[string]any
	if err := json.Unmarshal(buf.Bytes(), &rec); err != nil {
		t.Fatalf("log output is not one JSON object: %v\n%s", err, buf)
	}
	return rec
}

func TestTypesKeepSecretsOutOfTheLog(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	// 沒有 ReplaceAttr：這個測試只驗證型別自己的 LogValue。
	if rec := bind(wallet.NewHandler(slog.New(slog.NewJSONHandler(&buf, nil)))); rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	for _, s := range secrets {
		if strings.Contains(buf.String(), s) {
			t.Errorf("log contains %q:\n%s", s, buf.String())
		}
	}

	// 除錯需要的欄位還在。
	got := parse(t, &buf)
	req := got["request"].(map[string]any)
	if req["user_id"] != "u-42" || req["password"] != "[REDACTED]" {
		t.Errorf("request = %v", req)
	}
	if last4 := req["card"].(map[string]any)["last4"]; last4 != "3456" {
		t.Errorf("card.last4 = %v, want 3456", last4)
	}
	headers := got["headers"].(map[string]any)
	if headers["Content-Type"] != "application/json" || headers["Authorization"] != "[REDACTED]" {
		t.Errorf("headers = %v", headers)
	}
}

func TestBeforeLeaksTheSameRequest(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	bind(before.NewHandler(slog.New(slog.NewJSONHandler(&buf, nil))))
	for _, s := range secrets {
		if !strings.Contains(buf.String(), s) {
			t.Errorf("before log does not contain %q; the test above proves nothing", s)
		}
	}
}

// ANCHOR_END: test

func TestRedactMasksByKeyName(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{ReplaceAttr: wallet.Redact}))
	logger.Info("careless",
		slog.String("password", "example-not-a-real-password"),
		slog.Group("headers", slog.String("Authorization", "Bearer example-not-a-real-token")),
		slog.String("card_number", "1234567890123456"),
		slog.String("user_id", "u-42"))

	for _, s := range secrets {
		if strings.Contains(buf.String(), s) {
			t.Errorf("log contains %q:\n%s", s, buf.String())
		}
	}
	got := parse(t, &buf)
	if got["password"] != "[REDACTED]" || got["card_number"] != "[REDACTED]" || got["user_id"] != "u-42" {
		t.Errorf("record = %v", got)
	}
	if got["headers"].(map[string]any)["Authorization"] != "[REDACTED]" {
		t.Errorf("record = %v", got)
	}
}

func TestCardKeepsLastFour(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ in, want string }{
		{"1234567890123456", "3456"},
		{"1234-5678-9012-3456", "3456"},
		{"123", ""},
		{"", ""},
	} {
		var buf bytes.Buffer
		slog.New(slog.NewJSONHandler(&buf, nil)).Info("m", slog.Any("card", wallet.Card(tc.in)))
		got := parse(t, &buf)["card"]
		if tc.want == "" {
			if got != "[REDACTED]" {
				t.Errorf("card %q logged as %v, want [REDACTED]", tc.in, got)
			}
			continue
		}
		if got.(map[string]any)["last4"] != tc.want {
			t.Errorf("card %q logged as %v, want last4 %s", tc.in, got, tc.want)
		}
	}
}

func TestHeadersAreAllowListed(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	h := http.Header{}
	h.Set("Content-Type", "application/json")
	h.Set("X-Api-Key", "example-not-a-real-key") // 不在允許清單上，哪怕沒人預料到它
	slog.New(slog.NewJSONHandler(&buf, nil)).Info("m", slog.Any("headers", wallet.Headers(h)))
	if strings.Contains(buf.String(), "example-not-a-real-key") {
		t.Fatalf("log contains the key:\n%s", &buf)
	}
	headers := parse(t, &buf)["headers"].(map[string]any)
	if headers["Content-Type"] != "application/json" || headers["X-Api-Key"] != "[REDACTED]" {
		t.Errorf("headers = %v", headers)
	}
}
