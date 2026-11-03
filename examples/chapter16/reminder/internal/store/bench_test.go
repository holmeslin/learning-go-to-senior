package store_test

import (
	"path/filepath"
	"testing"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter16/reminder/internal/store"
)

// ANCHOR: bench

// BenchmarkBeginAndMarkSent 量一則提醒在資料層的成本：寫寄送中的紀錄，再記為已寄出，兩次寫入各自提交。
// 資料庫是檔案，和服務一樣開 WAL：提交要寫到檔案，記憶體資料庫量不到這一段。
// 準備工作在 b.Loop 的第一次呼叫之前，不計時；每一圈用新的報名者，Begin 才會真的寫入。
func BenchmarkBeginAndMarkSent(b *testing.B) {
	db, err := store.OpenDB(filepath.Join(b.TempDir(), "bench.db"), 4)
	if err != nil {
		b.Fatalf("open database: %v", err)
	}
	b.Cleanup(func() {
		if closeErr := db.Close(); closeErr != nil {
			b.Errorf("close database: %v", closeErr)
		}
	})
	if err = store.Migrate(b.Context(), db); err != nil {
		b.Fatalf("migrate: %v", err)
	}
	s := store.New(db)

	var registrant int64
	for b.Loop() {
		registrant++
		r := store.Reminder{EventID: 1, RegistrantID: registrant, DueAt: due}
		if ok, err := s.Begin(b.Context(), r, due); err != nil || !ok {
			b.Fatalf("Begin() = %v, %v", ok, err)
		}
		if err = s.MarkSent(b.Context(), r, due); err != nil {
			b.Fatalf("MarkSent() error = %v", err)
		}
	}
}

// ANCHOR_END: bench
