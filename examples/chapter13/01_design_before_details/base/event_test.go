package signup_test

import (
	"errors"
	"slices"
	"testing"

	signup "github.com/holmeslin/learning-go-to-senior/examples/chapter13/01_design_before_details/base"
)

func TestRegisterUntilFull(t *testing.T) {
	ev := signup.NewEvent(2)
	for _, who := range []string{"u1", "u2"} {
		if err := ev.Register(who); err != nil {
			t.Fatalf("Register(%q) = %v, want nil", who, err)
		}
	}
	if err := ev.Register("u3"); !errors.Is(err, signup.ErrFull) {
		t.Errorf("Register(u3) = %v, want ErrFull", err)
	}
	if got, want := ev.Attendees(), []string{"u1", "u2"}; !slices.Equal(got, want) {
		t.Errorf("Attendees() = %v, want %v", got, want)
	}
	if got, want := ev.Capacity(), 2; got != want {
		t.Errorf("Capacity() = %d, want %d", got, want)
	}
}

func TestRegisterRejectsDuplicateAndEmpty(t *testing.T) {
	ev := signup.NewEvent(2)
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
	ev := signup.NewEvent(1)
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
