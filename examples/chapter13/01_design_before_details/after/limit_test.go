package signup_test

import (
	"errors"
	"testing"

	signup "github.com/holmeslin/learning-go-to-senior/examples/chapter13/01_design_before_details/after"
)

// R5：候補上限是建構時的參數，每場活動各有自己的值，不會互相影響。
func TestWaitlistLimitIsPerEvent_R5(t *testing.T) {
	small := signup.NewEvent(1, &fakeNotifier{}, 1)
	large := signup.NewEvent(1, &fakeNotifier{}, 2)
	for _, ev := range []*signup.Event{small, large} {
		join(t, ev, "u1")
		join(t, ev, "u2")
	}

	if err := small.Register("u3"); !errors.Is(err, signup.ErrFull) {
		t.Errorf("small.Register(u3) = %v, want ErrFull", err)
	}
	if err := large.Register("u3"); !errors.Is(err, signup.ErrWaitlisted) {
		t.Errorf("large.Register(u3) = %v, want ErrWaitlisted", err)
	}
	if err := large.Register("u4"); !errors.Is(err, signup.ErrFull) {
		t.Errorf("large.Register(u4) = %v, want ErrFull", err)
	}
}
