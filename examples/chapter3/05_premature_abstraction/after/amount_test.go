package amount_test

import (
	"math"
	"testing"

	amount "github.com/holmeslin/learning-go-to-senior/examples/chapter3/05_premature_abstraction/after"
)

// ANCHOR: test

// 每個呼叫端各有一欄，欄裡的預期值只屬於那個呼叫端：
// 改收據的格式，不會讓帳務報表的測試失敗。
var cases = []struct {
	name    string
	cents   int64
	csv     string
	receipt string
	ledger  string
	invoice string
}{
	{"positive", 123450, "1234.50", "$1,234.50", "1,234.50", "$1,234.50"},
	{"negative", -1200, "-12.00", "-$12.00", "(12.00)", "($12.00)"},
	{"zero", 0, "0.00", "$0.00", "-", "$0.00"},
	{"small negative", -50, "-0.50", "-$0.50", "(0.50)", "($0.50)"},
	{"million", 100000000, "1000000.00", "$1,000,000.00", "1,000,000.00", "$1,000,000.00"},
}

func TestCallers(t *testing.T) {
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			check(t, "CSV", amount.CSV(tc.cents), tc.csv)
			check(t, "Receipt", amount.Receipt(tc.cents), tc.receipt)
			check(t, "Ledger", amount.Ledger(tc.cents), tc.ledger)
			check(t, "Invoice", amount.Invoice(tc.cents), tc.invoice)
		})
	}
}

func check(t *testing.T, caller, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %q; want %q", caller, got, want)
	}
}

// ANCHOR_END: test

// 共用的部分（拆出正負號、整數與小數）對每個呼叫端都一樣，邊角只需要處理一次。
func TestMinInt64(t *testing.T) {
	check(t, "CSV", amount.CSV(math.MinInt64), "-92233720368547758.08")
	check(t, "Invoice", amount.Invoice(math.MinInt64), "($92,233,720,368,547,758.08)")
}
