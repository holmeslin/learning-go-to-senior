package report_test

import (
	"fmt"
	"os"
	"strings"

	report "github.com/holmeslin/learning-go-to-senior/examples/chapter5/06_golden_file/after"
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

// 格式被改壞時，golden file 的比對會指出第一個不同的行。
func Example_formatBreaks() {
	var buf strings.Builder
	if err := report.Render(&buf, basicReport()); err != nil {
		fmt.Println(err)
		return
	}
	want, err := os.ReadFile("testdata/basic.golden")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("intact: %q\n", firstDiff(buf.String(), string(want)))
	fmt.Println("collapsed:", firstDiff(collapse(buf.String()), string(want)))
	// Output:
	// intact: ""
	// collapsed: line 5:
	//  got: "Coffee beans 1kg 3 x 18.00 = 54.00"
	// want: "  Coffee beans 1kg             3 x    18.00 =     54.00"
}

// ANCHOR_END: example
