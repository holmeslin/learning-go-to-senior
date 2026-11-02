package reminder_test

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter16/reminder/internal/store"
)

// keepSpans 是 Shutdown 之後仍留著 span 的記憶體 exporter：tracetest.InMemoryExporter 的 Shutdown 會清空已收到的 span，
// 而要檢查的正是關機時送出了什麼。
type keepSpans struct{ *tracetest.InMemoryExporter }

func (keepSpans) Shutdown(context.Context) error { return nil }

// ANCHOR: flush

// 關機時送出最後一段追蹤：掃描與送出的 span 先留在批次處理器的緩衝裡（5 秒才送一次），
// 收到結束訊號之後，Run 在關資料庫之前把它們送到 exporter。
func TestServiceSendsTheLastSpansOnShutdown(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		exporter := keepSpans{tracetest.NewInMemoryExporter()}
		h := newHarness(t, nil)
		h.exporter = exporter
		h.addEvent(t, 1, h.start, 2)
		cancel, done := h.run(t)

		time.Sleep(time.Second) // 一批掃描做完，span 還在緩衝裡
		if got := len(exporter.GetSpans()); got != 0 {
			t.Fatalf("spans exported before shutdown = %d, want 0 (still buffered)", got)
		}
		stop(t, cancel, done)

		names := map[string]int{}
		for _, s := range exporter.GetSpans() {
			names[s.Name]++
		}
		if names["reminder.scan"] != 1 || names["reminder.send"] != 2 || len(names) != 2 {
			t.Errorf("spans exported on shutdown = %v, want one scan and two sends", names)
		}
	})
}

// ANCHOR_END: flush

// 日誌不記裝置 token：失敗、重試、放棄都走過一遍，整份日誌裡找不到任何一個 token。
func TestServiceLogsNeverContainDeviceTokens(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var logs bytes.Buffer
		h := newHarness(t, nil)
		h.logger = slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
		h.provider.respond = func(r *http.Request) int {
			if r.Header.Get("Idempotency-Key") == "reminder-1-1" {
				return http.StatusBadRequest // 永久失敗
			}
			return http.StatusServiceUnavailable // 重試，直到放棄
		}
		h.addEvent(t, 1, h.start, 3)
		cancel, done := h.run(t)

		time.Sleep(2 * time.Hour)
		stop(t, cancel, done)

		if !strings.Contains(logs.String(), "scan finished") {
			t.Fatalf("logs = %q, want the scan summaries", logs.String())
		}
		if strings.Contains(logs.String(), "token-") {
			t.Errorf("logs contain a device token:\n%s", logs.String())
		}
	})
}

// logSink 是把每筆日誌送進 channel 的 slog.Handler：測試靠日誌知道服務的狀態（監聽的位址、開始關機）。
type logSink struct{ records chan slog.Record }

func (logSink) Enabled(context.Context, slog.Level) bool { return true }
func (s logSink) Handle(_ context.Context, r slog.Record) error {
	s.records <- r
	return nil
}
func (s logSink) WithAttrs([]slog.Attr) slog.Handler { return s }
func (s logSink) WithGroup(string) slog.Handler      { return s }

// await 讀日誌，直到 messages 裡每一則都出現過，回傳它們的屬性（字串）。逾時代表服務沒有走到那一步。
func (s logSink) await(t *testing.T, messages ...string) map[string]map[string]string {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	seen := map[string]map[string]string{}
	for len(seen) < len(messages) {
		select {
		case r := <-s.records:
			if !slices.Contains(messages, r.Message) {
				continue
			}
			attrs := map[string]string{}
			r.Attrs(func(a slog.Attr) bool { attrs[a.Key] = a.Value.String(); return true })
			seen[r.Message] = attrs
		case <-ctx.Done():
			t.Fatalf("logs seen so far = %v, still waiting for %v", seen, messages)
		}
	}
	return seen
}

// get 送出 GET，回傳狀態碼與本文。
func get(t *testing.T, url string) (int, string) {
	t.Helper()
	// 不重用連線：伺服器關閉時，沒有閒置的連線留著。
	client := &http.Client{Transport: &http.Transport{DisableKeepAlives: true}, Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body)
}

// ANCHOR: http

// 真的伺服器：內部 API 與診斷端點各自在隨機 port 上服務。收到結束訊號之後，readiness 先變成未就緒，
// 但兩個伺服器都等到進行中的送出做完才關：這段時間 /healthz 與 API 還答得出來，之後連線被拒。
// 這個測試不在 bubble 裡（bubble 裡不能有真的 socket），用 channel 與日誌同步，沒有 time.Sleep。
func TestServiceServesHTTPAndTurnsUnreadyBeforeWaitingForInFlightSends(t *testing.T) {
	h := newHarness(t, map[string]string{"REMINDER_SEND_CONCURRENCY": "1"})
	h.cfg.APIAddr, h.cfg.DiagAddr = "127.0.0.1:0", "127.0.0.1:0"
	h.addEvent(t, 1, h.start, 2)
	arrived, gate := make(chan struct{}, 2), make(chan struct{})
	h.provider.arrived, h.provider.gate = arrived, gate // 第一個請求到達之後，等 gate 才回應
	sink := logSink{records: make(chan slog.Record, 64)}
	h.logger = slog.New(sink)
	cancel, done := h.run(t)

	started := sink.await(t, "api server listening", "diagnostic server listening", "reminder service started")
	apiURL := "http://" + started["api server listening"]["addr"]
	diagURL := "http://" + started["diagnostic server listening"]["addr"]
	select {
	case <-arrived:
	case <-time.After(10 * time.Second):
		t.Fatal("no request reached the provider")
	}

	if code, _ := get(t, diagURL+"/readyz"); code != http.StatusOK {
		t.Errorf("GET /readyz while running = %d, want 200", code)
	}
	wantCounts := `{"event_id":1,"counts":{"failed":0,"retrying":0,"sending":1,"sent":0}}` + "\n"
	if code, body := get(t, apiURL+"/v1/events/1/reminder-counts"); code != http.StatusOK || body != wantCounts {
		t.Errorf("GET reminder-counts = %d %q, want 200 %q: one reminder is in flight", code, body, wantCounts)
	}
	if code, body := get(t, diagURL+"/debug/metrics"); code != http.StatusOK || !strings.Contains(body, "reminder.overdue.unsent") {
		t.Errorf("GET /debug/metrics = %d %q, want 200 and the overdue gauge", code, body)
	}

	cancel() // 收到結束訊號；進行中的送出還卡在供應商
	sink.await(t, "shutting down")
	if code, _ := get(t, diagURL+"/readyz"); code != http.StatusServiceUnavailable {
		t.Errorf("GET /readyz after the signal = %d, want 503", code)
	}
	if code, _ := get(t, diagURL+"/healthz"); code != http.StatusOK {
		t.Errorf("GET /healthz after the signal = %d, want 200: the process is still alive", code)
	}
	if code, _ := get(t, apiURL+"/v1/events/1/reminder-counts"); code != http.StatusOK {
		t.Errorf("GET reminder-counts after the signal = %d, want 200: the API stays up until sends finish", code)
	}

	close(gate) // 供應商回應，進行中的送出做完，Run 才繼續關伺服器
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Run() error = %v, want nil", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Run() did not return after the in-flight send finished")
	}

	client := &http.Client{Transport: &http.Transport{DisableKeepAlives: true}}
	if resp, err := client.Get(diagURL + "/healthz"); err == nil {
		resp.Body.Close()
		t.Error("diagnostic server still answers after Run returned, want connection refused")
	}
	if got := h.state(t, 1, 1).State; got != store.StateSent {
		t.Errorf("in-flight reminder state = %q, want sent", got)
	}
	if _, err := h.store.Get(t.Context(), 1, 2); err == nil {
		t.Error("the second reminder has a record, want none: it never started")
	}
}

// ANCHOR_END: http
