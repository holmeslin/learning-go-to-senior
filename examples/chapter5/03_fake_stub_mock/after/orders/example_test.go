package orders_test

import (
	"context"
	"fmt"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter5/03_fake_stub_mock/after/orders"
)

// ANCHOR: example

func Example_refactorKeepsTestsGreen() {
	ctx := context.Background()
	for _, p := range placers() {
		store, mailer := newFakeStore(), &recordingMailer{}
		svc := orders.NewService(catalog(), store, mailer)

		got, err := p.place(svc, ctx, "ann@example.com", lines())

		fmt.Printf("%-11s total=%d saved=%d mails=%d err=%v\n",
			p.name, got.Total, len(store.saved), len(mailer.sent), err)
	}
	// Output:
	// Place       total=440 saved=1 mails=1 err=<nil>
	// PlaceCached total=440 saved=1 mails=1 err=<nil>
}

// ANCHOR_END: example
