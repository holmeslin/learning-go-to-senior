package demoreg_test

import (
	"testing"
	"time"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter16/reminder/internal/demoreg"
	"github.com/holmeslin/learning-go-to-senior/examples/chapter16/reminder/internal/store"
)

// Seed 灌進去的人都在唯讀檢視裡；活動已存在時整筆交易退回，不留下半場活動。
func TestSeedFillsTheViewAndRejectsADuplicateEvent(t *testing.T) {
	t.Parallel()
	db, err := store.OpenDB("file:"+t.Name()+"?mode=memory&cache=shared", 1)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := demoreg.Apply(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	startsAt := time.Date(2030, 6, 11, 9, 0, 0, 0, time.UTC)
	count := func(query string, args ...any) (n int) {
		t.Helper()
		if err := db.QueryRowContext(t.Context(), query, args...).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	if err := demoreg.Seed(t.Context(), db, 7, startsAt, 25); err != nil {
		t.Fatalf("Seed() error = %v", err)
	}
	if got := count(`SELECT count(*) FROM reminder_candidates WHERE event_id = 7 AND starts_at = ?`, startsAt.Unix()); got != 25 {
		t.Errorf("rows of the view = %d, want 25", got)
	}

	if err := demoreg.Seed(t.Context(), db, 7, startsAt, 5); err == nil {
		t.Error("Seed() of an existing event error = nil, want an error")
	}
	if got := count(`SELECT count(*) FROM registrations`); got != 25 {
		t.Errorf("registrations after the rejected Seed = %d, want still 25", got)
	}
}
