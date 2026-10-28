package report_test

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	report "github.com/holmeslin/learning-go-to-senior/examples/chapter5/06_golden_file/after"
)

// ANCHOR: update

var update = flag.Bool("update", false, "rewrite the golden files in testdata")

// ANCHOR_END: update

// ANCHOR: golden

// assertGolden 把 got 和 testdata/<name>.golden 比對；帶 -update 時先用 got 覆寫它。
func assertGolden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", name+".golden")
	if *update {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("update %s: %v", path, err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden file: %v (run go test -update to create it)", err)
	}
	if diff := firstDiff(got, string(want)); diff != "" {
		t.Errorf("output differs from %s at %s\nrun go test -update if the change is intended", path, diff)
	}
}

// ANCHOR_END: golden

// firstDiff 回報兩段文字第一個不同的行；完全相同時回傳空字串。
func firstDiff(got, want string) string {
	g, w := strings.Split(got, "\n"), strings.Split(want, "\n")
	for i := 0; i < len(g) || i < len(w); i++ {
		if gl, wl := lineAt(g, i), lineAt(w, i); gl != wl {
			return fmt.Sprintf("line %d:\n got: %q\nwant: %q", i+1, gl, wl)
		}
	}
	return ""
}

func lineAt(lines []string, i int) string {
	if i >= len(lines) {
		return "<end of output>"
	}
	return lines[i]
}

var generated = time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

// 每次呼叫都回傳新的值，測試之間不共用可變狀態。
func basicReport() report.Report {
	return report.Report{
		Title:     "Monthly sales report",
		Generated: generated,
		Sales: []report.Sale{
			{Category: "Snacks", Item: "Rice crackers", Qty: 12, Cents: 175},
			{Category: "Beverages", Item: "Coffee beans 1kg", Qty: 3, Cents: 1800},
			{Category: "Stationery", Item: "Notebook", Qty: 5, Cents: 320},
			{Category: "Beverages", Item: "Green tea", Qty: 10, Cents: 450},
		},
	}
}

func refundReport() report.Report {
	return report.Report{
		Title:     "Monthly sales report",
		Generated: generated,
		Sales: []report.Sale{
			{Category: "Beverages", Item: "Coffee beans 1kg", Qty: 2, Cents: 1800},
			{Category: "Beverages", Item: "Coffee beans 1kg (refund)", Qty: -1, Cents: 1800},
			{Category: "Stationery", Item: "Sticker", Qty: 1, Cents: 5},
		},
	}
}

// ANCHOR: test

func TestRenderGolden(t *testing.T) {
	tests := []struct {
		name   string
		report report.Report
	}{
		{"basic", basicReport()},
		{"refund", refundReport()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf strings.Builder
			if err := report.Render(&buf, tt.report); err != nil {
				t.Fatalf("Render() error = %v", err)
			}
			assertGolden(t, tt.name, buf.String())
		})
	}
}

// ANCHOR_END: test

// ANCHOR: inline

// 輸出只有幾行，直接寫在測試裡，讀測試就知道預期什麼。
func TestRenderEmpty(t *testing.T) {
	var buf strings.Builder
	err := report.Render(&buf, report.Report{Title: "Monthly sales report", Generated: generated})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	want := "Monthly sales report\n" +
		"Generated: 2026-03-01\n" +
		"-------------------------------------------------------\n" +
		"  (no sales)\n" +
		"-------------------------------------------------------\n" +
		"TOTAL                                              0.00\n"
	if got := buf.String(); got != want {
		t.Errorf("Render() =\n%s\nwant\n%s", got, want)
	}
}

// ANCHOR_END: inline
