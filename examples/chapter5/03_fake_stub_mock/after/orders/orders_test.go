package orders_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter5/03_fake_stub_mock/after/orders"
)

// ANCHOR: state

func TestPlaceSavesOrder(t *testing.T) {
	t.Parallel()
	for _, p := range placers() {
		t.Run(p.name, func(t *testing.T) {
			t.Parallel()
			store := newFakeStore()
			svc := orders.NewService(catalog(), store, &recordingMailer{})

			got, err := p.place(svc, t.Context(), "ann@example.com", lines())

			if err != nil {
				t.Fatalf("%s() error = %v", p.name, err)
			}
			if want := 440; got.Total != want {
				t.Errorf("%s() total = %d, want %d", p.name, got.Total, want)
			}
			if saved := store.saved[got.ID]; !reflect.DeepEqual(saved, got) {
				t.Errorf("saved order = %+v, want %+v", saved, got)
			}
		})
	}
}

// ANCHOR_END: state

func TestPlaceRejectsUnknownSKU(t *testing.T) {
	t.Parallel()
	for _, p := range placers() {
		t.Run(p.name, func(t *testing.T) {
			t.Parallel()
			store, mailer := newFakeStore(), &recordingMailer{}
			svc := orders.NewService(catalog(), store, mailer)

			_, err := p.place(svc, t.Context(), "ann@example.com", []orders.Line{{SKU: "durian", Qty: 1}})

			if !errors.Is(err, orders.ErrUnknownSKU) {
				t.Errorf("%s() error = %v, want %v", p.name, err, orders.ErrUnknownSKU)
			}
			if len(store.saved) != 0 || len(mailer.sent) != 0 {
				t.Errorf("%s() saved %d orders and sent %d mails, want none", p.name, len(store.saved), len(mailer.sent))
			}
		})
	}
}

// ANCHOR: interaction

func TestPlaceSendsOneConfirmation(t *testing.T) {
	t.Parallel()
	for _, p := range placers() {
		t.Run(p.name, func(t *testing.T) {
			t.Parallel()
			mailer := &recordingMailer{}
			svc := orders.NewService(catalog(), newFakeStore(), mailer)

			got, err := p.place(svc, t.Context(), "ann@example.com", lines())

			if err != nil {
				t.Fatalf("%s() error = %v", p.name, err)
			}
			if len(mailer.sent) != 1 {
				t.Fatalf("%s() sent %d mails, want 1: %s", p.name, len(mailer.sent), subjects(mailer.sent))
			}
			if m := mailer.sent[0]; m.to != "ann@example.com" || !strings.Contains(m.subject, got.ID) {
				t.Errorf("mail = {to: %q, subject: %q}, want to ann@example.com and mention %q", m.to, m.subject, got.ID)
			}
		})
	}
}

func TestPlaceSendsNothingWhenSaveFails(t *testing.T) {
	t.Parallel()
	errDisk := errors.New("disk full")
	for _, p := range placers() {
		t.Run(p.name, func(t *testing.T) {
			t.Parallel()
			mailer := &recordingMailer{}
			svc := orders.NewService(catalog(), failingStore{err: errDisk}, mailer)

			_, err := p.place(svc, t.Context(), "ann@example.com", lines())

			if !errors.Is(err, errDisk) {
				t.Errorf("%s() error = %v, want it to wrap %v", p.name, err, errDisk)
			}
			if len(mailer.sent) != 0 {
				t.Errorf("%s() sent %d mails after a failed save, want 0: %s", p.name, len(mailer.sent), subjects(mailer.sent))
			}
		})
	}
}

// ANCHOR_END: interaction
