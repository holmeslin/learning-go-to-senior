package ledger_test

import (
	"testing"
	"time"

	ledger "github.com/holmeslin/learning-go-to-senior/examples/chapter3/04_generics_interface_or_copy/after"
)

func TestOrderTotalSkipsCancelled(t *testing.T) {
	orders := []ledger.Order{{Cents: 1200}, {Cents: 800, Cancelled: true}}
	if got := ledger.OrderTotal(orders); got != 1200 {
		t.Errorf("OrderTotal = %d; want 1200", got)
	}
}

// ANCHOR: rules

// 訂單用「取消」旗標、退款用「核准」旗標，零值的意思剛好相反：
// 這就是兩份迴圈長得像、卻不能共用同一條規則的原因。
func TestRefundTotalCountsApprovedOnly(t *testing.T) {
	refunds := []ledger.Refund{{Cents: 300, Approved: true}, {Cents: 200}}
	if got := ledger.RefundTotal(refunds); got != 300 {
		t.Errorf("RefundTotal = %d; want 300", got)
	}
	if got := ledger.OrderTotal([]ledger.Order{{Cents: 200}}); got != 200 {
		t.Errorf("OrderTotal of a zero-flag order = %d; want 200", got)
	}
	if got := ledger.RefundTotal([]ledger.Refund{{Cents: 200}}); got != 0 {
		t.Errorf("RefundTotal of a zero-flag refund = %d; want 0", got)
	}
}

// ANCHOR_END: rules

// Sum 的約束只有一行，不需要任何包裝：呼叫端的型別進去什麼、出來還是什麼。
func TestSumKeepsElementType(t *testing.T) {
	var d time.Duration = ledger.Sum([]time.Duration{time.Second, 2 * time.Second})
	if d != 3*time.Second {
		t.Errorf("Sum of durations = %v; want 3s", d)
	}
	var f float64 = ledger.Sum([]float64{0.5, 0.25})
	if f != 0.75 {
		t.Errorf("Sum of floats = %v; want 0.75", f)
	}
	if got := ledger.Sum[int](nil); got != 0 {
		t.Errorf("Sum of nothing = %d; want 0", got)
	}
}
