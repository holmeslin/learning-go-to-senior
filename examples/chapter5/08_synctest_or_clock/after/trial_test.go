package retry_test

import (
	"testing"
	"time"

	retry "github.com/holmeslin/learning-go-to-senior/examples/chapter5/08_synctest_or_clock/after"
)

// ANCHOR: trial

func TestTrialActive(t *testing.T) {
	taipei := time.FixedZone("CST", 8*3600)
	end := time.Date(2026, 12, 31, 12, 0, 0, 0, taipei) // 試用到 12/31 當天結束
	tests := []struct {
		name string
		now  time.Time
		want bool
	}{
		{"最後一天的早上", time.Date(2026, 12, 31, 8, 0, 0, 0, taipei), true},
		{"當天最後一秒", time.Date(2026, 12, 31, 23, 59, 59, 0, taipei), true},
		{"跨日的那一刻", time.Date(2027, 1, 1, 0, 0, 0, 0, taipei), false},
		{"同一刻換成 UTC 的寫法", time.Date(2026, 12, 31, 16, 0, 0, 0, time.UTC), false},
		{"時鐘倒退回試用期內", time.Date(2026, 1, 1, 0, 0, 0, 0, taipei), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trial := retry.Trial{End: end, Now: func() time.Time { return tt.now }}
			if got := trial.Active(); got != tt.want {
				t.Errorf("Active() at %v = %t, want %t", tt.now, got, tt.want)
			}
		})
	}
}

// ANCHOR_END: trial
