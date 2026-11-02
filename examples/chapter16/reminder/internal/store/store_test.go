package store_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter16/reminder/internal/demoreg"
	"github.com/holmeslin/learning-go-to-senior/examples/chapter16/reminder/internal/store"
)

// start 是測試裡所有活動的開始時間；預定時間是 start 往前 24 小時。
var start = time.Date(2030, 3, 2, 12, 0, 0, 0, time.UTC)

// due 是 start 對應的預定時間。
var due = start.Add(-store.LeadTime)

// ANCHOR: newstore

// newStore 給每個測試一個全新的記憶體資料庫，套用提醒服務的 migration 與示範用的報名 schema。
// 資料庫名稱由測試名稱決定，共享快取讓連線池裡的每條連線看到同一份資料。
func newStore(t *testing.T) (*store.Store, *sql.DB) {
	t.Helper()
	name := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := store.OpenDB("file:"+name+"?mode=memory&cache=shared", 4)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := db.Close(); closeErr != nil {
			t.Errorf("close database: %v", closeErr)
		}
	})
	if err = store.Migrate(t.Context(), db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err = demoreg.Apply(t.Context(), db); err != nil {
		t.Fatalf("apply demo registration schema: %v", err)
	}
	return store.New(db), db
}

// ANCHOR_END: newstore

// addEvent 新增一場開始時間為 startsAt 的活動，並讓每位報名者各有一筆報名。
func addEvent(t *testing.T, db *sql.DB, eventID int64, startsAt time.Time, registrants ...int64) {
	t.Helper()
	if err := demoreg.AddEvent(t.Context(), db, eventID, startsAt); err != nil {
		t.Fatal(err)
	}
	for _, id := range registrants {
		err := demoreg.AddRegistration(t.Context(), db, demoreg.Registration{
			EventID: eventID, RegistrantID: id, DeviceToken: fmt.Sprintf("token-%d", id), Email: "x@example.com",
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func mustDue(t *testing.T, s *store.Store, now time.Time) []store.Reminder {
	t.Helper()
	got, err := s.Due(t.Context(), now)
	if err != nil {
		t.Fatalf("Due(%v) error = %v", now, err)
	}
	return got
}

func mustGet(t *testing.T, s *store.Store, eventID, registrantID int64) store.Delivery {
	t.Helper()
	d, err := s.Get(t.Context(), eventID, registrantID)
	if err != nil {
		t.Fatalf("Get(%d, %d) error = %v", eventID, registrantID, err)
	}
	return d
}

// ANCHOR: begin

func TestBeginWritesTheRecordOnlyOnce(t *testing.T) {
	t.Parallel()
	s, db := newStore(t)
	addEvent(t, db, 1, start, 1)

	r := mustDue(t, s, due)[0]
	first, err := s.Begin(t.Context(), r, due)
	if err != nil || !first {
		t.Fatalf("first Begin() = %v, %v, want true, nil", first, err)
	}
	got := mustGet(t, s, 1, 1)
	if got.State != store.StateSending || got.Attempts != 1 || !got.DueAt.Equal(due) {
		t.Errorf("record after Begin = %+v, want sending, 1 attempt, due at %v", got, due)
	}

	// 唯一限制擋下第二次：寫不進去的人不送。
	second, err := s.Begin(t.Context(), r, due)
	if err != nil || second {
		t.Errorf("second Begin() = %v, %v, want false, nil", second, err)
	}
	if again := mustGet(t, s, 1, 1); again != got {
		t.Errorf("record after second Begin = %+v, want unchanged %+v", again, got)
	}
}

// ANCHOR_END: begin

func TestDueSelectsOnlyReminderThatAreDue(t *testing.T) {
	t.Parallel()
	s, db := newStore(t)
	addEvent(t, db, 1, start, 1, 2)
	// 候補者不在檢視裡。
	if err := demoreg.AddRegistration(t.Context(), db, demoreg.Registration{EventID: 1, RegistrantID: 3, DeviceToken: "w", Email: "w@example.com", Waitlisted: true}); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		now  time.Time
		want int
	}{
		{"one second before the due time", due.Add(-time.Second), 0},
		{"at the due time", due, 2},
		{"just before the send window closes", due.Add(store.SendWindow - time.Second), 2},
		{"at the end of the send window", due.Add(store.SendWindow), 0},
		{"after the event has started", start, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mustDue(t, s, tt.now); len(got) != tt.want {
				t.Errorf("Due(%v) = %d reminders, want %d", tt.now, len(got), tt.want)
			}
		})
	}
}

func TestDueSkipsRecordedAndReturnsRetriesThatAreDue(t *testing.T) {
	t.Parallel()
	s, db := newStore(t)
	addEvent(t, db, 1, start, 1, 2, 3)
	ctx := t.Context()

	all := mustDue(t, s, due)
	if len(all) != 3 {
		t.Fatalf("Due() = %d reminders, want 3", len(all))
	}
	// 1 號寄出、2 號待重試（十分鐘後）、3 號沒有紀錄。
	for _, r := range all[:2] {
		if ok, err := s.Begin(ctx, r, due); err != nil || !ok {
			t.Fatalf("Begin(%d) = %v, %v", r.RegistrantID, ok, err)
		}
	}
	retryAt := due.Add(10 * time.Minute)
	if err := s.MarkSent(ctx, all[0], due); err != nil {
		t.Fatal(err)
	}
	if err := s.MarkRetry(ctx, all[1], retryAt, "provider returned 503", due); err != nil {
		t.Fatal(err)
	}

	if got := mustDue(t, s, due.Add(time.Minute)); len(got) != 1 || got[0].RegistrantID != 3 {
		t.Errorf("Due() before the retry time = %+v, want only registrant 3", got)
	}
	got := mustDue(t, s, retryAt)
	if len(got) != 2 {
		t.Fatalf("Due() at the retry time = %+v, want the retry and registrant 3", got)
	}
	if got[0].RegistrantID != 2 || got[0].Attempts != 1 || got[1].RegistrantID != 3 || got[1].Attempts != 0 {
		t.Errorf("Due() = %+v, want registrant 2 (1 attempt so far) then registrant 3 (none)", got)
	}
}

// ANCHOR: transitions

func TestStateTransitions(t *testing.T) {
	t.Parallel()
	s, db := newStore(t)
	addEvent(t, db, 1, start, 1)
	ctx := t.Context()
	r := mustDue(t, s, due)[0]

	if ok, err := s.Begin(ctx, r, due); err != nil || !ok {
		t.Fatalf("Begin() = %v, %v", ok, err)
	}
	retryAt := due.Add(5 * time.Minute)
	if err := s.MarkRetry(ctx, r, retryAt, "provider returned 503", due); err != nil {
		t.Fatalf("MarkRetry() error = %v", err)
	}
	got := mustGet(t, s, 1, 1)
	if got.State != store.StateRetrying || !got.NextAttemptAt.Equal(retryAt) || got.LastError != "provider returned 503" {
		t.Fatalf("after MarkRetry = %+v, want retrying at %v with the cause", got, retryAt)
	}

	// 第二次嘗試：回到寄送中，次數加一。
	retry := mustDue(t, s, retryAt)[0]
	if ok, err := s.Begin(ctx, retry, retryAt); err != nil || !ok {
		t.Fatalf("Begin() for the retry = %v, %v", ok, err)
	}
	if got = mustGet(t, s, 1, 1); got.State != store.StateSending || got.Attempts != 2 || !got.NextAttemptAt.IsZero() {
		t.Fatalf("after the second Begin = %+v, want sending, 2 attempts, no next attempt", got)
	}
	if err := s.MarkSent(ctx, retry, retryAt); err != nil {
		t.Fatalf("MarkSent() error = %v", err)
	}
	if got = mustGet(t, s, 1, 1); got.State != store.StateSent {
		t.Errorf("after MarkSent state = %q, want sent", got.State)
	}
}

// ANCHOR_END: transitions

func TestMarkFailedAndTransitionsFromOtherStates(t *testing.T) {
	t.Parallel()
	s, db := newStore(t)
	addEvent(t, db, 1, start, 1)
	ctx := t.Context()
	r := mustDue(t, s, due)[0]

	// 還沒有 Begin：不在寄送中，不能直接記結果。
	if err := s.MarkSent(ctx, r, due); !errors.Is(err, store.ErrStateChanged) {
		t.Errorf("MarkSent() before Begin error = %v, want ErrStateChanged", err)
	}
	if ok, err := s.Begin(ctx, r, due); err != nil || !ok {
		t.Fatalf("Begin() = %v, %v", ok, err)
	}
	if err := s.MarkFailed(ctx, r, "provider returned 400", due); err != nil {
		t.Fatalf("MarkFailed() error = %v", err)
	}
	if got := mustGet(t, s, 1, 1); got.State != store.StateFailed || got.LastError != "provider returned 400" {
		t.Errorf("after MarkFailed = %+v, want failed with the cause", got)
	}
	// 失敗是終點，不會被後來的結果蓋掉。
	if err := s.MarkSent(ctx, r, due); !errors.Is(err, store.ErrStateChanged) {
		t.Errorf("MarkSent() after failure error = %v, want ErrStateChanged", err)
	}
}

// ANCHOR: expire

func TestFailOverdue(t *testing.T) {
	t.Parallel()
	s, db := newStore(t)
	// 1、2、3 號各有紀錄，4 號從未被挑出。
	addEvent(t, db, 1, start, 1, 2, 3, 4)
	ctx := t.Context()
	all := mustDue(t, s, due)
	for _, r := range all[:3] {
		if ok, err := s.Begin(ctx, r, due); err != nil || !ok {
			t.Fatalf("Begin(%d) = %v, %v", r.RegistrantID, ok, err)
		}
	}
	// 1 號已寄出；2 號停在寄送中（例如行程中斷）；3 號待重試。
	if err := s.MarkSent(ctx, all[0], due); err != nil {
		t.Fatal(err)
	}
	if err := s.MarkRetry(ctx, all[2], due.Add(10*time.Minute), "provider returned 503", due); err != nil {
		t.Fatal(err)
	}

	deadline := due.Add(store.SendWindow)
	expired, missed, err := s.FailOverdue(ctx, deadline.Add(-time.Second))
	if err != nil || expired != 0 || missed != 0 {
		t.Fatalf("FailOverdue() before the deadline = %d, %d, %v, want 0, 0, nil", expired, missed, err)
	}

	expired, missed, err = s.FailOverdue(ctx, deadline)
	if err != nil || expired != 2 || missed != 1 {
		t.Fatalf("FailOverdue() at the deadline = %d, %d, %v, want 2, 1, nil", expired, missed, err)
	}
	wantState := map[int64]store.State{1: store.StateSent, 2: store.StateFailed, 3: store.StateFailed, 4: store.StateFailed}
	for id, want := range wantState {
		if got := mustGet(t, s, 1, id).State; got != want {
			t.Errorf("registrant %d state = %q, want %q", id, got, want)
		}
	}
	// 沒有嘗試過就逾期的那一筆：次數是 0，原因寫在紀錄裡，不會再被挑出。
	if got := mustGet(t, s, 1, 4); got.Attempts != 0 || got.LastError == "" {
		t.Errorf("never-attempted record = %+v, want 0 attempts and a cause", got)
	}
	if got := mustDue(t, s, deadline); len(got) != 0 {
		t.Errorf("Due() after FailOverdue = %+v, want none", got)
	}
	if expired, missed, err = s.FailOverdue(ctx, deadline); err != nil || expired != 0 || missed != 0 {
		t.Errorf("second FailOverdue() = %d, %d, %v, want 0, 0, nil", expired, missed, err)
	}
}

// ANCHOR_END: expire

func TestFailOverdueIgnoresEventsThatAlreadyStarted(t *testing.T) {
	t.Parallel()
	s, db := newStore(t)
	addEvent(t, db, 1, start, 1)
	_, missed, err := s.FailOverdue(t.Context(), start.Add(time.Hour))
	if err != nil || missed != 0 {
		t.Errorf("FailOverdue() after the event started = %d, %v, want 0, nil", missed, err)
	}
}

func TestQueriesAreParameterized(t *testing.T) {
	t.Parallel()
	s, db := newStore(t)
	const token = `x'); DROP TABLE reminder_deliveries; --`
	if err := demoreg.AddEvent(t.Context(), db, 1, start); err != nil {
		t.Fatal(err)
	}
	err := demoreg.AddRegistration(t.Context(), db, demoreg.Registration{EventID: 1, RegistrantID: 1, DeviceToken: token, Email: "x@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	r := mustDue(t, s, due)[0]
	if r.DeviceToken != token {
		t.Fatalf("DeviceToken = %q, want it unchanged", r.DeviceToken)
	}
	if ok, err := s.Begin(t.Context(), r, due); err != nil || !ok {
		t.Fatalf("Begin() = %v, %v", ok, err)
	}
	if err := s.MarkFailed(t.Context(), r, token, due); err != nil {
		t.Fatalf("MarkFailed() error = %v", err)
	}
	if got := mustGet(t, s, 1, 1); got.LastError != token {
		t.Errorf("LastError = %q, want it stored as plain text", got.LastError)
	}
}

func TestGetUnknownDelivery(t *testing.T) {
	t.Parallel()
	s, _ := newStore(t)
	if _, err := s.Get(t.Context(), 1, 1); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("Get() error = %v, want ErrNotFound", err)
	}
}

func TestQueriesHonorTheContext(t *testing.T) {
	t.Parallel()
	s, db := newStore(t)
	addEvent(t, db, 1, start, 1)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := s.Due(ctx, due); !errors.Is(err, context.Canceled) {
		t.Errorf("Due() error = %v, want context.Canceled", err)
	}
	if _, _, err := s.FailOverdue(ctx, due); !errors.Is(err, context.Canceled) {
		t.Errorf("FailOverdue() error = %v, want context.Canceled", err)
	}
}

func TestCheckCandidates(t *testing.T) {
	t.Parallel()
	s, db := newStore(t)
	if err := s.CheckCandidates(t.Context()); err != nil {
		t.Errorf("CheckCandidates() with the view error = %v, want nil", err)
	}

	// 契約的欄位被改掉：檢查要在啟動時就發現，而不是等到第一次掃描。
	for _, stmt := range []string{
		`DROP VIEW reminder_candidates`,
		`CREATE VIEW reminder_candidates AS SELECT event_id, registrant_id, device_token, starts_at FROM registrations JOIN events ON events.id = event_id`,
	} {
		if _, err := db.ExecContext(t.Context(), stmt); err != nil {
			t.Fatal(err)
		}
	}
	err := s.CheckCandidates(t.Context())
	if err == nil || !strings.Contains(err.Error(), "reminder_candidates") {
		t.Errorf("CheckCandidates() without the email column error = %v, want it to name the view", err)
	}
}

func TestMigrateCanRunAgain(t *testing.T) {
	t.Parallel()
	_, db := newStore(t)
	if err := store.Migrate(t.Context(), db); err != nil {
		t.Errorf("second Migrate() error = %v, want nil", err)
	}
	var n int
	if err := db.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM reminder_migrations`).Scan(&n); err != nil || n != 1 {
		t.Errorf("applied migrations = %d, %v, want 1, nil", n, err)
	}
}
