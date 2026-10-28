package logstats_test

import (
	"fmt"
	"strings"
	"testing"

	logstats "github.com/holmeslin/learning-go-to-senior/examples/chapter7/03_pprof_cpu_or_heap/after"
	before "github.com/holmeslin/learning-go-to-senior/examples/chapter7/03_pprof_cpu_or_heap/before"
)

// ANCHOR: same

// 改寫不能改變結果：同一批輸入，before 與 after 的彙總逐字相同。
func TestSummarizeMatchesBefore(t *testing.T) {
	inputs := map[string][]string{
		"一般流量":  sampleLines(500),
		"沒有任何行": nil,
		"超長路徑":  {"2026-10-28T10:00:00Z GET /api/" + strings.Repeat("a", 10000) + "/42 200 3"},
	}
	for name, lines := range inputs {
		t.Run(name, func(t *testing.T) {
			want, err := before.Summarize(lines)
			if err != nil {
				t.Fatal(err)
			}
			got, err := logstats.Summarize(lines)
			if err != nil {
				t.Fatal(err)
			}
			if fmt.Sprint(got) != fmt.Sprint(want) {
				t.Errorf("Summarize() = %v, want %v", got, want)
			}
		})
	}
}

// ANCHOR_END: same

func TestSummarizeErrors(t *testing.T) {
	tests := []struct {
		name  string
		lines []string
		want  string
	}{
		{"欄位不足", []string{"2026-10-28T10:00:00Z GET /a 200 1", "2026-10-28T10:00:01Z GET /a"}, "line 2: malformed line"},
		{"時間格式錯誤", []string{"yesterday GET /a 200 1"}, `line 1: parsing time "yesterday" as "2006-01-02T15:04:05Z07:00": cannot parse "yesterday" as "2006"`},
		{"延遲不是數字", []string{"2026-10-28T10:00:00Z GET /a 200 fast"}, `line 1: strconv.Atoi: parsing "fast": invalid syntax`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := logstats.Summarize(tt.lines)
			if err == nil || err.Error() != tt.want {
				t.Errorf("Summarize() error = %v, want %s", err, tt.want)
			}
		})
	}
}

// ANCHOR: allocs

// 每行不再配置：2000 行只配置幾十次（before 是 4057 次，每行至少 2 次）。
// 剩下的配置是 5 個路徑各一個 ms slice 的倍增擴張（約 10 次）加上 map 與緩衝區，
// 隨行數只有對數成長。這條路徑不經過 sync.Pool，次數是確定的，-race 下也一樣；
// 上限 100 比實測的 65 寬一些，Go 調整 slice 擴張策略時不會誤報。
func TestSummarizeAllocsDoNotScaleWithLines(t *testing.T) {
	lines := sampleLines(2000)
	allocs := testing.AllocsPerRun(10, func() {
		if _, err := logstats.Summarize(lines); err != nil {
			t.Fatal(err)
		}
	})
	if allocs > 100 {
		t.Errorf("Summarize() allocs = %v for %d lines, want <= 100", allocs, len(lines))
	}
}

// ANCHOR_END: allocs
