package orders_test

import (
	"testing"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter5/03_fake_stub_mock/before/orders"
)

// ANCHOR: test

// wantCalls 是 Place 照現在的寫法會做的每一件事，依序列出。
func wantCalls() []string {
	return []string{
		"Price(apple)", "Price(pear)", "Price(apple)",
		"Save(ann@example.com, total=440)",
		`Send(ann@example.com, "order o-1 confirmed")`,
	}
}

func TestPlace(t *testing.T) {
	t.Parallel()
	m := &mock{}
	svc := orders.NewService(m, m, m)
	lines := []orders.Line{{SKU: "apple", Qty: 2}, {SKU: "pear", Qty: 1}, {SKU: "apple", Qty: 1}}

	if _, err := svc.Place(t.Context(), "ann@example.com", lines); err != nil {
		t.Fatalf("Place() error = %v", err)
	}

	if err := m.verify(wantCalls()...); err != nil {
		t.Error(err)
	}
}

// ANCHOR_END: test
