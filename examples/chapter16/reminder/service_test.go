package reminder_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter16/reminder"
	"github.com/holmeslin/learning-go-to-senior/examples/chapter16/reminder/internal/config"
	"github.com/holmeslin/learning-go-to-senior/examples/chapter16/reminder/internal/demoreg"
	"github.com/holmeslin/learning-go-to-senior/examples/chapter16/reminder/internal/store"
)

// ANCHOR: provider

// request 是假供應商收到的一次呼叫。
type request struct {
	Key string        // Idempotency-Key 標頭
	At  time.Duration // 距離測試開始多久
}

// provider 是假的推播供應商，同時是 http.RoundTripper：不開 socket，記錄每次呼叫，
// 回應由 respond 決定。delay 是每個請求占用的假時間；請求的 context 結束時，不等 delay 就回傳。
type provider struct {
	start   time.Time
	delay   time.Duration
	respond func(r *http.Request) int // 回傳 HTTP 狀態碼；nil 代表 200
	// arrived 與 gate 給不在 bubble 裡的測試用（真的時間）：每個請求到達時先通知 arrived，
	// 再等 gate 關閉才回應；兩個都是 nil 代表不等。
	arrived chan<- struct{}
	gate    <-chan struct{}

	mu       sync.Mutex
	requests []request
}

func (p *provider) RoundTrip(r *http.Request) (*http.Response, error) {
	p.mu.Lock()
	p.requests = append(p.requests, request{Key: r.Header.Get("Idempotency-Key"), At: time.Since(p.start)})
	p.mu.Unlock()

	if p.arrived != nil {
		p.arrived <- struct{}{}
	}
	if p.gate != nil {
		select {
		case <-p.gate:
		case <-r.Context().Done():
			return nil, r.Context().Err()
		}
	}
	if p.delay > 0 {
		select {
		case <-time.After(p.delay):
		case <-r.Context().Done():
			return nil, r.Context().Err()
		}
	}
	code := http.StatusOK
	if p.respond != nil {
		code = p.respond(r)
	}
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
}

// count 回傳每個冪等鍵被呼叫了幾次。
func (p *provider) count() map[string]int {
	p.mu.Lock()
	defer p.mu.Unlock()
	counts := map[string]int{}
	for _, r := range p.requests {
		counts[r.Key]++
	}
	return counts
}

// firstAt 回傳某個冪等鍵第一次被呼叫的時間。
func (p *provider) firstAt(key string) time.Duration {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, r := range p.requests {
		if r.Key == key {
			return r.At
		}
	}
	return -1
}

func (p *provider) total() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.requests)
}

// ANCHOR_END: provider

// ANCHOR: harness

// harness 是 bubble 裡跑起來的整個服務：真的設定、真的 sqlite（記憶體）、假的供應商，
// 測試只從外部觀察它：供應商收到什麼、寄送紀錄寫了什麼、Run 什麼時候回傳。
type harness struct {
	cfg      config.Config
	db       *sql.DB // 測試自己的連線：灌報名資料、讀寄送紀錄
	store    *store.Store
	provider *provider
	start    time.Time

	logger   *slog.Logger          // 服務用的日誌；預設丟棄
	exporter sdktrace.SpanExporter // 追蹤送去的地方；預設 nil（不送出）
}

// loadConfig 用 config.Load 載入設定：預設的環境變數是開關打開、供應商網址、每個測試各自的記憶體資料庫，
// overrides 蓋過它們。
func loadConfig(t *testing.T, overrides map[string]string) config.Config {
	t.Helper()
	name := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	vars := map[string]string{
		"REMINDER_SEND_ENABLED": "true",
		"REMINDER_PROVIDER_URL": "http://provider.test",
		"REMINDER_DB_PATH":      "file:" + name + "?mode=memory&cache=shared",
	}
	maps.Copy(vars, overrides)
	cfg, err := config.Load(func(key string) string { return vars[key] })
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
	// 空位址代表不啟動 HTTP 伺服器：synctest 的 bubble 裡不能有真的 socket。
	// config.Load 不會產生空位址（空字串視為沒設，會用預設值），所以在這裡直接清掉；
	// 要測真的伺服器的測試，在 bubble 外自己設成 127.0.0.1:0。
	cfg.APIAddr, cfg.DiagAddr = "", ""
	return cfg
}

// newHarness 載入設定、開資料庫並建立報名的示範 schema。資料庫在 bubble 裡開，就在 bubble 裡關。
func newHarness(t *testing.T, overrides map[string]string) *harness {
	t.Helper()
	cfg := loadConfig(t, overrides)
	db, err := store.OpenDB(cfg.DatabasePath, 2)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := demoreg.Apply(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	return &harness{cfg: cfg, db: db, store: store.New(db), provider: &provider{start: start}, start: start}
}

// addEvent 新增一場預定時間在 due 的活動，報名者編號是 1 到 registrants。
func (h *harness) addEvent(t *testing.T, eventID int64, due time.Time, registrants int) {
	t.Helper()
	if err := demoreg.AddEvent(t.Context(), h.db, eventID, due.Add(store.LeadTime)); err != nil {
		t.Fatal(err)
	}
	for id := range registrants {
		err := demoreg.AddRegistration(t.Context(), h.db, demoreg.Registration{
			EventID: eventID, RegistrantID: int64(id + 1), DeviceToken: fmt.Sprintf("token-%d-%d", eventID, id+1), Email: "x@example.com",
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

// run 在背景啟動服務。cancel 代表「收到結束訊號」；Run 的結果從 done 讀。
func (h *harness) run(t *testing.T) (cancel context.CancelFunc, done <-chan error) {
	t.Helper()
	logger := h.logger
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	svc := &reminder.Service{Config: h.cfg, Logger: logger, Transport: h.provider, SpanExporter: h.exporter}
	ctx, cancel := context.WithCancel(t.Context())
	result := make(chan error, 1)
	go func() { result <- svc.Run(ctx) }()
	return cancel, result
}

// state 回傳一筆寄送紀錄；找不到就讓測試失敗。
func (h *harness) state(t *testing.T, eventID, registrantID int64) store.Delivery {
	t.Helper()
	d, err := h.store.Get(t.Context(), eventID, registrantID)
	if err != nil {
		t.Fatalf("Get(%d, %d) error = %v", eventID, registrantID, err)
	}
	return d
}

// ANCHOR_END: harness

// stop 結束服務並檢查它正常回傳。
func stop(t *testing.T, cancel context.CancelFunc, done <-chan error) {
	t.Helper()
	cancel()
	if err := <-done; err != nil {
		t.Errorf("Run() error = %v, want nil", err)
	}
}

// ANCHOR: once

// 每則到期的提醒恰好送出一次：兩場活動、五則提醒，其中一場的預定時間在 30 分鐘後。
func TestServiceSendsEachDueReminderOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, nil)
		h.addEvent(t, 1, h.start, 3)
		h.addEvent(t, 2, h.start.Add(30*time.Minute), 2)
		cancel, done := h.run(t)

		time.Sleep(time.Second) // 第一次掃描做完；限流讓三則分散在 40ms 之內
		if got := h.provider.total(); got != 3 {
			t.Fatalf("requests in the first scan = %d, want 3 (the second event is not due yet)", got)
		}

		time.Sleep(2 * time.Hour) // 之後的每一次掃描都不再送已經寄出的
		counts := h.provider.count()
		if len(counts) != 5 {
			t.Fatalf("distinct reminders sent = %d, want 5: %v", len(counts), counts)
		}
		for key, n := range counts {
			if n != 1 {
				t.Errorf("reminder %s sent %d times, want exactly once", key, n)
			}
		}
		if at := h.provider.firstAt("reminder-2-1"); at < 30*time.Minute || at > 30*time.Minute+time.Second {
			t.Errorf("event 2 first sent at %v, want 30m: the scan at its due time", at)
		}
		for event, registrants := range map[int64]int64{1: 3, 2: 2} {
			for id := int64(1); id <= registrants; id++ {
				if got := h.state(t, event, id).State; got != store.StateSent {
					t.Errorf("reminder %d/%d state = %q, want sent", event, id, got)
				}
			}
		}
		stop(t, cancel, done)
	})
}

// ANCHOR_END: once

// ANCHOR: giveup

// 供應商持續失敗：預定時間後 1 小時記為失敗，之後不再送。
func TestServiceGivesUpAfterOneHour(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, nil)
		h.provider.respond = func(*http.Request) int { return http.StatusServiceUnavailable }
		h.addEvent(t, 1, h.start, 2)
		cancel, done := h.run(t)

		time.Sleep(59 * time.Minute)
		synctest.Wait()
		if got := h.state(t, 1, 1); got.State != store.StateRetrying || got.Attempts < 2 {
			t.Errorf("after 59m: %+v, want still retrying after more than one attempt", got)
		}

		time.Sleep(2 * time.Minute) // 61 分鐘：預定時間後 1 小時的那次掃描已經過了
		synctest.Wait()
		sentBefore := h.provider.total()
		for id := int64(1); id <= 2; id++ {
			got := h.state(t, 1, id)
			if got.State != store.StateFailed || !got.UpdatedAt.Equal(h.start.Add(store.SendWindow)) {
				t.Errorf("registrant %d after 61m: %+v, want failed at the 1h scan", id, got)
			}
		}

		time.Sleep(30 * time.Minute) // 記為失敗之後，再多掃幾次也不再送
		if got := h.provider.total(); got != sentBefore {
			t.Errorf("requests grew from %d to %d after giving up, want no more", sentBefore, got)
		}
		stop(t, cancel, done)
	})
}

// ANCHOR_END: giveup

// ANCHOR: disabled

// 開關關閉：服務照常啟動，但不掃描、不呼叫供應商、不寫任何紀錄。
func TestServiceDoesNotSendWhenSwitchIsOff(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, map[string]string{"REMINDER_SEND_ENABLED": "false"})
		h.addEvent(t, 1, h.start, 3)
		cancel, done := h.run(t)

		time.Sleep(time.Hour)
		if got := h.provider.total(); got != 0 {
			t.Errorf("requests = %d, want none while the switch is off", got)
		}
		var n int
		if err := h.db.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM reminder_deliveries`).Scan(&n); err != nil || n != 0 {
			t.Errorf("delivery records = %d, %v, want none: the old worker's reminders must stay untouched", n, err)
		}
		stop(t, cancel, done)
	})
}

// ANCHOR_END: disabled

// ANCHOR: shutdown

// 取消之後：進行中的送出做完，Run 在關機期限內回傳；沒有開始的提醒不留紀錄。
// 「沒有殘留的 goroutine」由 synctest.Test 保證：bubble 裡還有 goroutine 沒結束，測試就失敗。
func TestServiceStopsWithinShutdownTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, map[string]string{"REMINDER_SEND_CONCURRENCY": "2"})
		h.provider.delay = 3 * time.Second // 每個請求在路上三秒
		h.addEvent(t, 1, h.start, 5)
		cancel, done := h.run(t)

		time.Sleep(time.Second) // 兩個請求在路上
		cancelledAt := time.Now()
		cancel()
		err := <-done
		elapsed := time.Since(cancelledAt)

		if err != nil {
			t.Errorf("Run() error = %v, want nil", err)
		}
		// 兩個請求各在路上三秒（限流讓第二個晚 20ms 出發），從第 1 秒收到訊號起算，約 2 秒後做完。
		if elapsed < 2*time.Second || elapsed > h.cfg.ShutdownTimeout {
			t.Errorf("Run() returned %v after the signal, want about 2s (the in-flight requests), within %v", elapsed, h.cfg.ShutdownTimeout)
		}
		if got := h.provider.total(); got != 2 {
			t.Errorf("requests = %d, want 2: reminders not started before the signal stay unsent", got)
		}
		for id := int64(1); id <= 2; id++ {
			if got := h.state(t, 1, id).State; got != store.StateSent {
				t.Errorf("in-flight reminder %d state = %q, want sent", id, got)
			}
		}
		for id := int64(3); id <= 5; id++ {
			if _, err := h.store.Get(t.Context(), 1, id); !errors.Is(err, store.ErrNotFound) {
				t.Errorf("reminder %d: Get() error = %v, want ErrNotFound", id, err)
			}
		}
	})
}

// ANCHOR_END: shutdown

// 期限到了仍有送出沒結束：Run 回傳錯誤。這裡的設定刻意繞過 config.Load 的檢查，
// 把關機期限設得比請求逾時短；Load 會拒絕這樣的設定。
func TestServiceReportsAnErrorWhenShutdownDeadlinePasses(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, map[string]string{"REMINDER_REQUEST_TIMEOUT": "30s", "REMINDER_SHUTDOWN_TIMEOUT": "60s"})
		h.provider.delay = time.Hour // 供應商不回應，直到請求逾時
		h.cfg.ShutdownTimeout = 2 * time.Second
		h.addEvent(t, 1, h.start, 1)
		cancel, done := h.run(t)

		synctest.Wait()
		cancelledAt := time.Now()
		cancel()
		err := <-done
		if !errors.Is(err, context.DeadlineExceeded) || time.Since(cancelledAt) != 2*time.Second {
			t.Errorf("Run() = %v after %v, want DeadlineExceeded after 2s", err, time.Since(cancelledAt))
		}
		time.Sleep(time.Minute) // 殘留的請求在自己的逾時之後結束，bubble 才能收掉
	})
}

// ANCHOR: missed

// 服務停機超過 1 小時：預定時間後 1 小時內一次都沒送過，第一次掃描就直接記為失敗，不送出。
func TestServiceRecordsAsFailedWhatItMissedWhileDown(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, nil)
		h.addEvent(t, 1, h.start.Add(-90*time.Minute), 2) // 預定時間在 90 分鐘前
		cancel, done := h.run(t)

		synctest.Wait()
		if got := h.provider.total(); got != 0 {
			t.Errorf("requests = %d, want none: the send window already closed", got)
		}
		for id := int64(1); id <= 2; id++ {
			if got := h.state(t, 1, id); got.State != store.StateFailed || got.Attempts != 0 {
				t.Errorf("registrant %d: %+v, want failed with 0 attempts", id, got)
			}
		}
		stop(t, cancel, done)
	})
}

// ANCHOR_END: missed

// 報名服務的唯讀檢視不在：服務啟動時就拒絕，不是等到第一次掃描才出錯；開關關閉的部署也一樣。
func TestServiceRefusesToStartWithoutTheView(t *testing.T) {
	for _, enabled := range []string{"true", "false"} {
		t.Run("switch "+enabled, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				svc := &reminder.Service{
					Config: loadConfig(t, map[string]string{"REMINDER_SEND_ENABLED": enabled}),
					Logger: slog.New(slog.DiscardHandler),
				}
				err := svc.Run(t.Context())
				if err == nil || !strings.Contains(err.Error(), "reminder_candidates") {
					t.Errorf("Run() error = %v, want it to refuse to start and name the view", err)
				}
			})
		})
	}
}
