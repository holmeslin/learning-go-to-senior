package dispatch_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter16/reminder/demoreg"
	"github.com/holmeslin/learning-go-to-senior/examples/chapter16/reminder/dispatch"
	"github.com/holmeslin/learning-go-to-senior/examples/chapter16/reminder/push"
	"github.com/holmeslin/learning-go-to-senior/examples/chapter16/reminder/store"
)

// ANCHOR: sender

// call 是假供應商收到的一次呼叫。
type call struct {
	Key string
	At  time.Time
}

// fakeSender 記錄每次呼叫；behave 決定回傳什麼（nil 代表成功），delay 是每次呼叫占用的（假）時間。
// 呼叫發生在 synctest 的 bubble 裡，所以 delay 用 time.Sleep 也不會真的等。
type fakeSender struct {
	delay  time.Duration
	behave func(n push.Notification) error

	mu       sync.Mutex
	calls    []call
	inflight int
	peak     int
}

func (f *fakeSender) Send(_ context.Context, n push.Notification) error {
	f.mu.Lock()
	f.calls = append(f.calls, call{Key: n.Key, At: time.Now()})
	f.inflight++
	f.peak = max(f.peak, f.inflight)
	f.mu.Unlock()

	time.Sleep(f.delay)

	f.mu.Lock()
	f.inflight--
	f.mu.Unlock()
	if f.behave != nil {
		return f.behave(n)
	}
	return nil
}

func (f *fakeSender) snapshot() []call {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]call(nil), f.calls...)
}

// ANCHOR_END: sender

// ANCHOR: setup

// setup 在 bubble 裡開一個記憶體資料庫，放進一場「現在就到預定時間」的活動與 n 位報名者。
// 資料庫在 bubble 裡開，就要在 bubble 裡關。
func setup(t *testing.T, n int) (*store.Store, *sql.DB) {
	t.Helper()
	name := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := store.OpenDB("file:"+name+"?mode=memory&cache=shared", 4)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err = store.Migrate(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	if err = demoreg.Apply(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	addEvent(t, db, 1, time.Now().Add(store.LeadTime), n)
	return store.New(db), db
}

// ANCHOR_END: setup

func addEvent(t *testing.T, db *sql.DB, eventID int64, startsAt time.Time, registrants int) {
	t.Helper()
	if err := demoreg.AddEvent(t.Context(), db, eventID, startsAt); err != nil {
		t.Fatal(err)
	}
	for i := range registrants {
		err := demoreg.AddRegistration(t.Context(), db, demoreg.Registration{
			EventID: eventID, RegistrantID: int64(i + 1), DeviceToken: fmt.Sprintf("token-%d", i+1), Email: "x@example.com",
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func options() dispatch.Options {
	return dispatch.Options{
		Interval:      5 * time.Minute,
		Concurrency:   5,
		RatePerSecond: math.MaxFloat64, // 不限流；要測限流的測試自己設
		QueryTimeout:  time.Second,
		Logger:        slog.New(slog.DiscardHandler),
	}
}

func newDispatcher(t *testing.T, st *store.Store, sender dispatch.Sender, opts dispatch.Options) *dispatch.Dispatcher {
	t.Helper()
	d, err := dispatch.New(st, sender, opts)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func mustScan(t *testing.T, d *dispatch.Dispatcher) dispatch.Summary {
	t.Helper()
	sum, err := d.Scan(t.Context())
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	return sum
}

func state(t *testing.T, st *store.Store, registrantID int64) store.Delivery {
	t.Helper()
	d, err := st.Get(t.Context(), 1, registrantID)
	if err != nil {
		t.Fatalf("Get(1, %d) error = %v", registrantID, err)
	}
	return d
}

// ANCHOR: once

func TestScanSendsEachDueReminderOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		st, _ := setup(t, 3)
		sender := &fakeSender{}
		d := newDispatcher(t, st, sender, options())

		if sum := mustScan(t, d); sum.Sent != 3 {
			t.Fatalf("first Scan() = %+v, want 3 sent", sum)
		}
		// 再掃一次：已經有紀錄的不會再被挑出。
		if sum := mustScan(t, d); sum != (dispatch.Summary{}) {
			t.Errorf("second Scan() = %+v, want nothing to do", sum)
		}

		keys := map[string]int{}
		for _, c := range sender.snapshot() {
			keys[c.Key]++
		}
		if len(keys) != 3 {
			t.Errorf("sent keys = %v, want 3 distinct keys", keys)
		}
		for key, n := range keys {
			if n != 1 {
				t.Errorf("key %s sent %d times, want once", key, n)
			}
		}
		for id := int64(1); id <= 3; id++ {
			if got := state(t, st, id).State; got != store.StateSent {
				t.Errorf("registrant %d state = %q, want sent", id, got)
			}
		}
	})
}

// ANCHOR_END: once

func TestScanWaitsUntilTheReminderIsDue(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		st, db := setup(t, 0)
		// 活動 30 小時後開始：預定時間在 6 小時後。
		addEvent(t, db, 2, time.Now().Add(30*time.Hour), 1)
		sender := &fakeSender{}
		d := newDispatcher(t, st, sender, options())

		if sum := mustScan(t, d); sum.Sent != 0 {
			t.Fatalf("Scan() 6 hours early = %+v, want nothing sent", sum)
		}
		time.Sleep(6 * time.Hour)
		if sum := mustScan(t, d); sum.Sent != 1 {
			t.Fatalf("Scan() at the due time = %+v, want 1 sent", sum)
		}
	})
}

// ANCHOR: concurrency

func TestScanLimitsConcurrency(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		st, _ := setup(t, 12)
		sender := &fakeSender{delay: time.Second}
		opts := options()
		opts.Concurrency = 5
		d := newDispatcher(t, st, sender, opts)

		start := time.Now()
		if sum := mustScan(t, d); sum.Sent != 12 {
			t.Fatalf("Scan() = %+v, want 12 sent", sum)
		}
		if sender.peak != 5 {
			t.Errorf("peak concurrent sends = %d, want 5", sender.peak)
		}
		// 12 則、一次 5 則：5 + 5 + 2，三輪，每輪一秒。
		if got, want := time.Since(start), 3*time.Second; got != want {
			t.Errorf("Scan() took %v, want %v", got, want)
		}
	})
}

// ANCHOR_END: concurrency

// ANCHOR: ratelimit

func TestScanRespectsTheRateLimit(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		st, _ := setup(t, 100)
		sender := &fakeSender{} // 呼叫不占時間：瓶頸只剩限流
		opts := options()
		opts.Concurrency = 20
		opts.RatePerSecond = 50
		d := newDispatcher(t, st, sender, opts)

		start := time.Now()
		if sum := mustScan(t, d); sum.Sent != 100 {
			t.Fatalf("Scan() = %+v, want 100 sent", sum)
		}
		// 並行上限是 20，但每秒只放行 50 則：第一則立刻送，之後每 20ms 一則，第 100 則在 1.98s。
		calls := sender.snapshot()
		last := calls[len(calls)-1].At.Sub(start)
		if last < 1970*time.Millisecond || last > 1990*time.Millisecond {
			t.Errorf("last send at %v, want about 1.98s", last)
		}
		for i := 1; i < len(calls); i++ {
			if gap := calls[i].At.Sub(calls[i-1].At); gap < 19*time.Millisecond {
				t.Fatalf("sends %d and %d are %v apart, want at least 20ms", i-1, i, gap)
			}
		}
	})
}

// ANCHOR_END: ratelimit

// ANCHOR: retry

func TestRunRetriesWithBackoffThenGivesUp(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		st, _ := setup(t, 1)
		start := time.Now() // 活動的預定時間就是現在
		sender := &fakeSender{behave: func(push.Notification) error {
			return fmt.Errorf("provider returned 503: %w", push.ErrTransient)
		}}
		opts := options()
		opts.Jitter = func(limit time.Duration) time.Duration { return limit / 2 } // 固定取一半，時間點才斷言得出來
		d := newDispatcher(t, st, sender, opts)

		ctx, cancel := context.WithCancel(t.Context())
		done := make(chan struct{})
		go func() { defer close(done); d.Run(ctx) }()
		time.Sleep(2 * time.Hour)
		cancel()
		<-done

		// 退避的上限 5m、10m、20m、30m、30m…，各取一半，但重試只發生在每 5 分鐘一次的掃描：
		// 0 失敗 → 2m30s 後可重試 → 5m 的掃描；5m 失敗 → 5m 後 → 10m；10m 失敗 → 10m 後 → 20m；
		// 20m 失敗 → 15m 後 → 35m；35m 失敗 → 15m 後 → 50m；50m 失敗 → 65m，已經超過期限。
		want := []time.Duration{0, 5 * time.Minute, 10 * time.Minute, 20 * time.Minute, 35 * time.Minute, 50 * time.Minute}
		var got []time.Duration
		keys := map[string]bool{}
		for _, c := range sender.snapshot() {
			got = append(got, c.At.Sub(start))
			keys[c.Key] = true
		}
		if fmt.Sprint(got) != fmt.Sprint(want) {
			t.Errorf("send times = %v, want %v", got, want)
		}
		if len(keys) != 1 {
			t.Errorf("keys used = %v, want the same key for every attempt", keys)
		}

		// 預定時間後 1 小時的那次掃描把它記為失敗，之後不再送。
		final := state(t, st, 1)
		if final.State != store.StateFailed || final.Attempts != 6 || !final.UpdatedAt.Equal(start.Add(store.SendWindow)) {
			t.Errorf("final record = %+v, want failed after 6 attempts at %v", final, start.Add(store.SendWindow))
		}
		if !strings.Contains(final.LastError, "503") {
			t.Errorf("LastError = %q, want the provider's last failure", final.LastError)
		}
	})
}

// ANCHOR_END: retry

func TestScanPermanentFailureIsNotRetried(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		st, _ := setup(t, 1)
		sender := &fakeSender{behave: func(push.Notification) error {
			return fmt.Errorf("provider returned 400: %w", push.ErrPermanent)
		}}
		d := newDispatcher(t, st, sender, options())

		if sum := mustScan(t, d); sum.Failed != 1 {
			t.Fatalf("Scan() = %+v, want 1 failed", sum)
		}
		time.Sleep(time.Minute)
		if sum := mustScan(t, d); sum != (dispatch.Summary{}) {
			t.Errorf("second Scan() = %+v, want nothing to do", sum)
		}
		if n := len(sender.snapshot()); n != 1 {
			t.Errorf("sender called %d times, want once", n)
		}
		if got := state(t, st, 1); got.State != store.StateFailed || got.Attempts != 1 {
			t.Errorf("record = %+v, want failed after 1 attempt", got)
		}
	})
}

func TestScanTreatsUnclassifiedErrorAsPermanent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		st, _ := setup(t, 1)
		sender := &fakeSender{behave: func(push.Notification) error { return errors.New("who knows") }}
		d := newDispatcher(t, st, sender, options())
		if sum := mustScan(t, d); sum.Failed != 1 || sum.Retrying != 0 {
			t.Errorf("Scan() = %+v, want 1 failed and no retry", sum)
		}
	})
}

// ANCHOR: missed

func TestScanRecordsMissedReminderAsFailedWithoutSending(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		st, db := setup(t, 0)
		// 預定時間在 90 分鐘前，一次都沒有送過（例如服務停機）：
		addEvent(t, db, 2, time.Now().Add(store.LeadTime-90*time.Minute), 2)
		sender := &fakeSender{}
		d := newDispatcher(t, st, sender, options())

		sum := mustScan(t, d)
		if sum.Missed != 2 || sum.Sent != 0 {
			t.Fatalf("Scan() = %+v, want 2 missed and nothing sent", sum)
		}
		if n := len(sender.snapshot()); n != 0 {
			t.Errorf("sender called %d times, want none", n)
		}
		got, err := st.Get(t.Context(), 2, 1)
		if err != nil || got.State != store.StateFailed || got.Attempts != 0 {
			t.Errorf("record = %+v, %v, want failed with 0 attempts", got, err)
		}
	})
}

// ANCHOR_END: missed

// ANCHOR: skipped

func TestScanSkipsReminderRecordedByAnotherWriter(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		st, _ := setup(t, 2)
		due := time.Now()
		sender := &fakeSender{}
		opts := options()
		opts.Concurrency = 1   // 一則一則來，第二則一定排在第一則之後
		opts.RatePerSecond = 1 // 第二則要等一秒才輪到
		d := newDispatcher(t, st, sender, opts)

		done := make(chan dispatch.Summary, 1)
		go func() {
			sum, _ := d.Scan(t.Context())
			done <- sum
		}()

		// 第一則已經送出、第二則還在等限流的時候，別人（例如報名服務裡的舊 worker）先寫了第二則的紀錄。
		time.Sleep(500 * time.Millisecond)
		second := store.Reminder{EventID: 1, RegistrantID: 2, DueAt: due}
		if ok, err := st.Begin(t.Context(), second, time.Now()); err != nil || !ok {
			t.Fatalf("Begin() by the other writer = %v, %v", ok, err)
		}

		sum := <-done
		if sum.Sent != 1 || sum.Skipped != 1 {
			t.Errorf("Scan() = %+v, want 1 sent and 1 skipped", sum)
		}
		if n := len(sender.snapshot()); n != 1 {
			t.Errorf("sender called %d times, want once: the unique key keeps the second one unsent", n)
		}
	})
}

// ANCHOR_END: skipped

// ANCHOR: cancel

func TestRunLetsInFlightSendFinishWhenCanceled(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		st, _ := setup(t, 3)
		release := make(chan struct{})
		var once sync.Once
		sender := &fakeSender{behave: func(push.Notification) error {
			once.Do(func() { <-release }) // 第一則卡住，直到測試放行
			return nil
		}}
		opts := options()
		opts.Concurrency = 1
		d := newDispatcher(t, st, sender, opts)

		ctx, cancel := context.WithCancel(t.Context())
		done := make(chan struct{})
		go func() { defer close(done); d.Run(ctx) }()
		synctest.Wait() // 第一則送出到一半

		cancel() // 收到結束訊號
		synctest.Wait()
		select {
		case <-done:
			t.Fatal("Run() returned while a send was still in flight")
		default:
		}

		close(release)
		<-done
		// 進行中的那一則做完、記為已寄出；其餘的沒有開始，也沒有留下紀錄。
		if got := state(t, st, 1).State; got != store.StateSent {
			t.Errorf("in-flight reminder state = %q, want sent", got)
		}
		for id := int64(2); id <= 3; id++ {
			if _, err := st.Get(t.Context(), 1, id); !errors.Is(err, store.ErrNotFound) {
				t.Errorf("registrant %d: Get() error = %v, want ErrNotFound (never started)", id, err)
			}
		}
		if n := len(sender.snapshot()); n != 1 {
			t.Errorf("sender called %d times, want once", n)
		}
	})
}

// ANCHOR_END: cancel

func TestNewRejectsInvalidOptions(t *testing.T) {
	t.Parallel()
	tests := map[string]func(*dispatch.Options){
		"no interval":      func(o *dispatch.Options) { o.Interval = 0 },
		"no concurrency":   func(o *dispatch.Options) { o.Concurrency = 0 },
		"no rate":          func(o *dispatch.Options) { o.RatePerSecond = 0 },
		"no query timeout": func(o *dispatch.Options) { o.QueryTimeout = 0 },
		"retry max below base": func(o *dispatch.Options) {
			o.RetryBase, o.RetryMax = time.Hour, time.Minute
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			opts := options()
			mutate(&opts)
			if _, err := dispatch.New(nil, &fakeSender{}, opts); err == nil {
				t.Error("New() error = nil, want an error")
			}
		})
	}
}
