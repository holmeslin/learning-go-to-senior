package signup_test

import (
	"errors"
	"slices"
	"testing"

	signup "github.com/holmeslin/learning-go-to-senior/examples/chapter13/01_design_before_details/after"
)

func TestRegisterUntilFull(t *testing.T) {
	ev := newEvent(2, &fakeNotifier{})
	for _, who := range []string{"u1", "u2"} {
		if err := ev.Register(who); err != nil {
			t.Fatalf("Register(%q) = %v, want nil", who, err)
		}
	}
	if got, want := ev.Confirmed(), []string{"u1", "u2"}; !slices.Equal(got, want) {
		t.Errorf("Confirmed() = %v, want %v", got, want)
	}
	if got, want := ev.Seats(), 2; got != want {
		t.Errorf("Seats() = %d, want %d", got, want)
	}
}

func TestRegisterRejectsDuplicateAndEmpty(t *testing.T) {
	ev := newEvent(2, &fakeNotifier{})
	if err := ev.Register("u1"); err != nil {
		t.Fatal(err)
	}
	if err := ev.Register("u1"); !errors.Is(err, signup.ErrAlreadyRegistered) {
		t.Errorf("second Register(u1) = %v, want ErrAlreadyRegistered", err)
	}
	if err := ev.Register("  "); !errors.Is(err, signup.ErrEmptyName) {
		t.Errorf("Register(blank) = %v, want ErrEmptyName", err)
	}
}

func TestCancelFreesSeat(t *testing.T) {
	ev := newEvent(1, &fakeNotifier{})
	if err := ev.Register("u1"); err != nil {
		t.Fatal(err)
	}
	if err := ev.Cancel("u1"); err != nil {
		t.Fatalf("Cancel(u1) = %v, want nil", err)
	}
	if err := ev.Cancel("u1"); !errors.Is(err, signup.ErrNotRegistered) {
		t.Errorf("second Cancel(u1) = %v, want ErrNotRegistered", err)
	}
	if err := ev.Register("u2"); err != nil {
		t.Errorf("Register(u2) after cancel = %v, want nil", err)
	}
}
