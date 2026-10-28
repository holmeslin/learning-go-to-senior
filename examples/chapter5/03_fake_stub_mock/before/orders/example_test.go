package orders_test

import (
	"context"
	"fmt"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter5/03_fake_stub_mock/before/orders"
)

// ANCHOR: example

func Example_refactorBreaksMock() {
	ctx := context.Background()
	lines := []orders.Line{{SKU: "apple", Qty: 2}, {SKU: "pear", Qty: 1}, {SKU: "apple", Qty: 1}}
	places := []struct {
		name  string
		place func(*orders.Service, context.Context, string, []orders.Line) (orders.Order, error)
	}{
		{"Place", (*orders.Service).Place},
		{"PlaceCached", (*orders.Service).PlaceCached},
	}

	for _, p := range places {
		m := &mock{}
		got, err := p.place(orders.NewService(m, m, m), ctx, "ann@example.com", lines)
		fmt.Printf("%s: total=%d err=%v\n", p.name, got.Total, err)
		fmt.Printf("mock verify: %v\n", m.verify(wantCalls()...))
	}
	// Output:
	// Place: total=440 err=<nil>
	// mock verify: <nil>
	// PlaceCached: total=440 err=<nil>
	// mock verify: calls differ
	// got:  Price(apple) Price(pear) Save(ann@example.com, total=440) Send(ann@example.com, "order o-1 confirmed")
	// want: Price(apple) Price(pear) Price(apple) Save(ann@example.com, total=440) Send(ann@example.com, "order o-1 confirmed")
}

// ANCHOR_END: example
