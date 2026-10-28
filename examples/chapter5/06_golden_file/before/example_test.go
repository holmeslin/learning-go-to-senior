package report_test

import (
	"fmt"
	"strings"

	report "github.com/holmeslin/learning-go-to-senior/examples/chapter5/06_golden_file/before"
)

// collapse 把每一行的連續空白縮成一個，模擬有人改壞了欄位對齊。
func collapse(s string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = strings.Join(strings.Fields(l), " ")
	}
	return strings.Join(lines, "\n")
}

// ANCHOR: example

// 欄位對齊被改壞的輸出，子字串檢查照樣通過。
func Example_substringCheck() {
	var buf strings.Builder
	if err := report.Render(&buf, basicReport()); err != nil {
		fmt.Println(err)
		return
	}
	broken := collapse(buf.String())
	fmt.Print(broken)
	fmt.Printf("missing: %q\n", hasParts(broken))
	// Output:
	// Monthly sales report
	// Generated: 2026-03-01
	// -------------------------------------------------------
	// Beverages
	// Coffee beans 1kg 3 x 18.00 = 54.00
	// Green tea 10 x 4.50 = 45.00
	// subtotal 99.00
	// Snacks
	// Rice crackers 12 x 1.75 = 21.00
	// subtotal 21.00
	// Stationery
	// Notebook 5 x 3.20 = 16.00
	// subtotal 16.00
	// -------------------------------------------------------
	// TOTAL 136.00
	// missing: ""
}

// ANCHOR_END: example
