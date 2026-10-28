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

// 配置次數不隨行數成長：每 10 行配置不到一次（before 每行至少配置一次）。
func TestSummarizeAllocsDoNotScaleWithLines(t *testing.T) {
	lines := sampleLines(2000)
	allocs := testing.AllocsPerRun(10, func() {
		if _, err := logstats.Summarize(lines); err != nil {
			t.Fatal(err)
		}
	})
	if limit := float64(len(lines)) / 10; allocs >= limit {
		t.Errorf("Summarize() allocs = %v for %d lines, want < %v", allocs, len(lines), limit)
	}
}

// ANCHOR_END: allocs
