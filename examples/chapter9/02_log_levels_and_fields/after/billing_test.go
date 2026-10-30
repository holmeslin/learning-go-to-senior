package billing_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	billing "github.com/holmeslin/learning-go-to-senior/examples/chapter9/02_log_levels_and_fields/after"
)

// ANCHOR: test

// decode 把 JSONHandler 寫進 buf 的每一行解析成一筆紀錄。
func decode(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()
	var records []map[string]any
	for dec := json.NewDecoder(buf); dec.More(); {
		var rec map[string]any
		if err := dec.Decode(&rec); err != nil {
			t.Fatal(err)
		}
		records = append(records, rec)
	}
	return records
}

func TestFailedRequestLogsOnce(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	h := billing.NewHandler(&billing.Service{Store: &billing.Store{Down: true}}, logger)

	for _, id := range []string{"r-1", "r-2"} {
		req := httptest.NewRequest("POST", "/invoices/inv-7/pay", nil)
		req.Header.Set("X-Request-ID", id)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", rec.Code)
		}
	}

	records := decode(t, &buf)
	if len(records) != 2 {
		t.Fatalf("got %d records, want 2 (one per failed request): %v", len(records), records)
	}
	for i, id := range []string{"r-1", "r-2"} {
		if records[i]["level"] != "ERROR" || records[i][billing.KeyRequestID] != id {
			t.Errorf("record %d = %v, want one ERROR for %s", i, records[i], id)
		}
	}
}

// ANCHOR_END: test

func TestLevelVarEnablesDebug(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	var level slog.LevelVar
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: &level}))
	h := billing.NewHandler(&billing.Service{Store: &billing.Store{}}, logger)
	pay := func() {
		req := httptest.NewRequest("POST", "/invoices/inv-7/pay", nil)
		req.Header.Set("X-Request-ID", "r-1")
		h.ServeHTTP(httptest.NewRecorder(), req)
	}

	pay()
	for _, rec := range decode(t, &buf) {
		if rec["level"] == "DEBUG" {
			t.Fatalf("DEBUG record at Info level: %v", rec)
		}
	}

	level.Set(slog.LevelDebug)
	pay()
	records := decode(t, &buf)
	if len(records) != 2 || records[0]["level"] != "DEBUG" {
		t.Fatalf("records = %v, want a DEBUG record then an INFO record", records)
	}
	// Debug 那一行沒有自己寫欄位，請求範圍的欄位是 With 帶進來的。
	if records[0][billing.KeyRequestID] != "r-1" || records[0][billing.KeyInvoiceID] != "inv-7" {
		t.Errorf("debug record lost request fields: %v", records[0])
	}
}

func TestKeysComeFromTheSharedSet(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	for _, down := range []bool{false, true} {
		h := billing.NewHandler(&billing.Service{Store: &billing.Store{Down: down}}, logger)
		req := httptest.NewRequest("POST", "/invoices/inv-7/pay", nil)
		req.Header.Set("X-Request-ID", "r-1")
		req.Header.Set("X-User-ID", "u-42")
		h.ServeHTTP(httptest.NewRecorder(), req)
	}
	billing.ImportInvoices(logger, []string{"inv-1", ""})

	allowed := []string{
		slog.TimeKey, slog.LevelKey, slog.MessageKey,
		billing.KeyRequestID, billing.KeyUserID, billing.KeyInvoiceID,
		billing.KeyError, billing.KeyDuration, billing.KeyImported, billing.KeySkipped,
	}
	for _, rec := range decode(t, &buf) {
		for key := range rec {
			if !slices.Contains(allowed, key) {
				t.Errorf("key %q in %v is not in the shared set", key, rec)
			}
		}
	}
}

func TestDuration(t *testing.T) {
	t.Parallel()
	a := billing.Duration(1500 * time.Millisecond)
	if a.Key != "duration_ms" || a.Value.Int64() != 1500 {
		t.Errorf("Duration(1.5s) = %v, want duration_ms=1500", a)
	}
}

func TestImportLevels(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		level     slog.Level
		ids       []string
		wantLevel []string
	}{
		{"clean batch at Info", slog.LevelInfo, []string{"a", "b", "c"}, []string{"INFO"}},
		{"skipped rows warn once", slog.LevelInfo, []string{"a", "", "c"}, []string{"WARN"}},
		{"debug adds one record per row", slog.LevelDebug, []string{"a", "", "c"}, []string{"DEBUG", "DEBUG", "DEBUG", "WARN"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: tt.level}))
			billing.ImportInvoices(logger, tt.ids)

			var got []string
			for _, rec := range decode(t, &buf) {
				got = append(got, rec["level"].(string))
			}
			if !slices.Equal(got, tt.wantLevel) {
				t.Errorf("levels = %v, want %v", got, tt.wantLevel)
			}
		})
	}
}
