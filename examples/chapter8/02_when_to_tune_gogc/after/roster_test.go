package roster_test

import (
	"bytes"
	"testing"

	roster "github.com/holmeslin/learning-go-to-senior/examples/chapter8/02_when_to_tune_gogc/after"
	before "github.com/holmeslin/learning-go-to-senior/examples/chapter8/02_when_to_tune_gogc/before"
)

// ANCHOR: same

// 改寫不能改變結果：同一批 ID（含不存在的、重複的、空批次），before 與 after 的報表逐位元組相同。
func TestHandleMatchesBefore(t *testing.T) {
	batches := map[string][]int{
		"一般批次":     {1, 5, 9, 42},
		"含不存在的 ID": {3, 999, 4, -1},
		"重複的 ID":   {7, 7, 7},
		"全部都不存在":   {500, 501},
		"空批次":      nil,
		"ID 位數不同":  {0, 9, 10, 99},
		"整份名冊":     {0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
	}
	old, now := before.Load(100), roster.Load(100)
	for name, ids := range batches {
		t.Run(name, func(t *testing.T) {
			want := old.Handle(ids)
			if got := now.Handle(ids); !bytes.Equal(got, want) {
				t.Errorf("Handle(%v) = %q, want %q", ids, got, want)
			}
		})
	}
}

// ANCHOR_END: same

// ANCHOR: allocs

// 每批只配置一次（回傳的報表），不隨 ID 的數量成長；before 每個 ID 至少配置一次。
// 這條路徑沒有 sync.Pool，次數是確定的，-race 下也一樣。
func TestHandleAllocsDoNotScaleWithBatchSize(t *testing.T) {
	r := roster.Load(1000)
	ids := make([]int, 500)
	for i := range ids {
		ids[i] = i
	}
	allocs := testing.AllocsPerRun(10, func() { r.Handle(ids) })
	if allocs > 1 {
		t.Errorf("Handle() allocs = %v for %d ids, want <= 1", allocs, len(ids))
	}
}

// ANCHOR_END: allocs
