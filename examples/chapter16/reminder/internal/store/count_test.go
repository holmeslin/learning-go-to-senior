package store_test

import (
	"errors"
	"maps"
	"testing"
	"time"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter16/reminder/internal/store"
)

// 各狀態的數量：四個狀態都在、沒有的是 0；只算這場活動。
func TestCountByState(t *testing.T) {
	t.Parallel()
	s, db := newStore(t)
	addEvent(t, db, 1, start, 1, 2, 3, 4)
	addEvent(t, db, 2, start, 1)
	ctx := t.Context()

	all := mustDue(t, s, due)
	if len(all) != 5 {
		t.Fatalf("Due() = %d reminders, want 5", len(all))
	}
	for _, r := range all {
		if ok, err := s.Begin(ctx, r, due); err != nil || !ok {
			t.Fatalf("Begin(%d/%d) = %v, %v", r.EventID, r.RegistrantID, ok, err)
		}
	}
	// 活動 1：兩個寄出、一個待重試、一個停在寄送中；活動 2：一個失敗。
	for _, r := range all[:2] {
		if err := s.MarkSent(ctx, r, due); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.MarkRetry(ctx, all[2], due.Add(5*time.Minute), "provider returned 503", due); err != nil {
		t.Fatal(err)
	}
	if err := s.MarkFailed(ctx, all[4], "provider returned 400", due); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		event int64
		want  map[store.State]int
	}{
		{1, map[store.State]int{store.StateSent: 2, store.StateRetrying: 1, store.StateSending: 1, store.StateFailed: 0}},
		{2, map[store.State]int{store.StateSent: 0, store.StateRetrying: 0, store.StateSending: 0, store.StateFailed: 1}},
	}
	for _, tt := range tests {
		got, err := s.CountByState(ctx, tt.event)
		if err != nil || !maps.Equal(got, tt.want) {
			t.Errorf("CountByState(%d) = %v, %v, want %v", tt.event, got, err, tt.want)
		}
	}
}

// 活動有報名者、還沒有任何紀錄：存在，四個狀態都是 0。檢視裡沒有這場活動：ErrNotFound。
func TestCountByStateDistinguishesAnEventWithoutRecordsFromAnUnknownOne(t *testing.T) {
	t.Parallel()
	s, db := newStore(t)
	addEvent(t, db, 1, start, 1)
	addEvent(t, db, 2, start) // 活動存在，但沒有報名者

	want := map[store.State]int{store.StateSent: 0, store.StateRetrying: 0, store.StateSending: 0, store.StateFailed: 0}
	if got, err := s.CountByState(t.Context(), 1); err != nil || !maps.Equal(got, want) {
		t.Errorf("CountByState(1) = %v, %v, want %v", got, err, want)
	}
	for _, id := range []int64{2, 99} {
		if _, err := s.CountByState(t.Context(), id); !errors.Is(err, store.ErrNotFound) {
			t.Errorf("CountByState(%d) error = %v, want ErrNotFound", id, err)
		}
	}
}

// 逾期未寄：已過預定時間、還沒寄出的。寄出的、失敗的、還沒到預定時間的都不算。
func TestCountOverdue(t *testing.T) {
	t.Parallel()
	s, db := newStore(t)
	addEvent(t, db, 1, start, 1, 2, 3, 4)
	ctx := t.Context()

	count := func(now time.Time) int {
		t.Helper()
		n, err := s.CountOverdue(ctx, now)
		if err != nil {
			t.Fatalf("CountOverdue(%v) error = %v", now, err)
		}
		return n
	}
	if got := count(due.Add(-time.Second)); got != 0 {
		t.Errorf("before the due time = %d, want 0", got)
	}
	if got := count(due); got != 4 {
		t.Errorf("at the due time, nothing attempted = %d, want 4", got)
	}

	all := mustDue(t, s, due)
	for _, r := range all {
		if ok, err := s.Begin(ctx, r, due); err != nil || !ok {
			t.Fatalf("Begin(%d) = %v, %v", r.RegistrantID, ok, err)
		}
	}
	if err := s.MarkSent(ctx, all[0], due); err != nil {
		t.Fatal(err)
	}
	if err := s.MarkFailed(ctx, all[1], "provider returned 400", due); err != nil {
		t.Fatal(err)
	}
	if err := s.MarkRetry(ctx, all[2], due.Add(5*time.Minute), "provider returned 503", due); err != nil {
		t.Fatal(err)
	}
	// 3 號待重試、4 號停在寄送中：兩筆還沒寄出。
	if got := count(due.Add(time.Minute)); got != 2 {
		t.Errorf("one sent, one failed, one retrying, one sending = %d, want 2", got)
	}
}
