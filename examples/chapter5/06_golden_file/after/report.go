// Package report 產生月銷售報表的純文字版本。
package report

import (
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"
	"time"
)

// Sale 是報表裡的一筆銷售。退貨用負的 Qty 表示。
type Sale struct {
	Category string // 分類，報表依分類名稱排序分組
	Item     string // 品名
	Qty      int    // 數量
	Cents    int    // 單價，單位是分
}

// Report 是 Render 的輸入。
type Report struct {
	Title     string
	Generated time.Time // 由呼叫端傳入，Render 不自己讀時鐘
	Sales     []Sale
}

const rule = "-------------------------------------------------------\n"

// Render 把 r 寫成純文字報表：依分類名稱排序、分類內維持輸入順序，
// 每個分類有小計，最後是總計。
func Render(w io.Writer, r Report) error {
	byCategory := map[string][]Sale{}
	for _, s := range r.Sales {
		byCategory[s.Category] = append(byCategory[s.Category], s)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s\nGenerated: %s\n%s", r.Title, r.Generated.Format("2006-01-02"), rule)
	total := 0
	if len(r.Sales) == 0 {
		b.WriteString("  (no sales)\n")
	}
	for _, category := range slices.Sorted(maps.Keys(byCategory)) {
		b.WriteString(category + "\n")
		subtotal := 0
		for _, s := range byCategory[category] {
			amount := s.Qty * s.Cents
			subtotal += amount
			fmt.Fprintf(&b, "  %-26s %3d x %8s = %9s\n", s.Item, s.Qty, money(s.Cents), money(amount))
		}
		fmt.Fprintf(&b, "%-28s %26s\n", "  subtotal", money(subtotal))
		total += subtotal
	}
	b.WriteString(rule)
	fmt.Fprintf(&b, "%-28s %26s\n", "TOTAL", money(total))

	_, err := io.WriteString(w, b.String())
	return err
}

func money(cents int) string {
	if cents < 0 {
		return "-" + money(-cents)
	}
	return fmt.Sprintf("%d.%02d", cents/100, cents%100)
}
