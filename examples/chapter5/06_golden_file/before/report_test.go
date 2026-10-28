package report_test

import (
	"strings"
	"testing"
	"time"

	report "github.com/holmeslin/learning-go-to-senior/examples/chapter5/06_golden_file/before"
)

// 每次呼叫都回傳新的值，測試之間不共用可變狀態。
func basicReport() report.Report {
	return report.Report{
		Title:     "Monthly sales report",
		Generated: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		Sales: []report.Sale{
			{Category: "Snacks", Item: "Rice crackers", Qty: 12, Cents: 175},
			{Category: "Beverages", Item: "Coffee beans 1kg", Qty: 3, Cents: 1800},
			{Category: "Stationery", Item: "Notebook", Qty: 5, Cents: 320},
			{Category: "Beverages", Item: "Green tea", Qty: 10, Cents: 450},
		},
	}
}

func render(t testing.TB, r report.Report) string {
	t.Helper()
	var buf strings.Builder
	if err := report.Render(&buf, r); err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	return buf.String()
}

// ANCHOR: exact

// 寫法一：整份預期輸出寫死在字串裡。
func TestRenderExact(t *testing.T) {
	want := `Monthly sales report
Generated: 2026-03-01
-------------------------------------------------------
Beverages
  Coffee beans 1kg             3 x    18.00 =     54.00
  Green tea                   10 x     4.50 =     45.00
  subtotal                                        99.00
Snacks
  Rice crackers               12 x     1.75 =     21.00
  subtotal                                        21.00
Stationery
  Notebook                     5 x     3.20 =     16.00
  subtotal                                        16.00
-------------------------------------------------------
TOTAL                                            136.00
`
	if got := render(t, basicReport()); got != want {
		t.Errorf("Render() =\n%q\nwant\n%q", got, want)
	}
}

// ANCHOR_END: exact

// ANCHOR: parts

// hasParts 檢查輸出裡有沒有該出現的字句，回傳缺少的第一個；都有就回傳空字串。
func hasParts(out string) string {
	for _, part := range []string{"Beverages", "Coffee beans 1kg", "subtotal", "TOTAL", "136.00"} {
		if !strings.Contains(out, part) {
			return part
		}
	}
	return ""
}

// 寫法二：只檢查幾個子字串。
func TestRenderParts(t *testing.T) {
	if missing := hasParts(render(t, basicReport())); missing != "" {
		t.Errorf("Render() output is missing %q", missing)
	}
}

// ANCHOR_END: parts
