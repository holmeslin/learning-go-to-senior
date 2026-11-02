package signup_test

import (
	"errors"
	"slices"
	"testing"
	"testing/synctest"

	signup "github.com/holmeslin/learning-go-to-senior/examples/chapter13/01_design_before_details/after"
)

// join 為 who 報名。排進候補（ErrWaitlisted）也算報名成功，其他錯誤讓測試失敗。
func join(t *testing.T, ev *signup.Event, who string) {
	t.Helper()
	if err := ev.Register(who); err != nil && !errors.Is(err, signup.ErrWaitlisted) {
		t.Fatalf("Register(%q) = %v", who, err)
	}
}

// ANCHOR: r1

// R1：遞補是 domain 的責任。直接對 Event 取消（不經過 HTTP handler），
// 也要遞補候補名單的第一位並通知他。
func TestCancelPromotesFirstWaitlisted_R1(t *testing.T) {
	n := &fakeNotifier{}
	ev := newEvent(1, n)
	join(t, ev, "u1")
	join(t, ev, "u2")
	join(t, ev, "u3")

	if err := ev.Cancel("u1"); err != nil {
		t.Fatal(err)
	}

	if got, want := ev.Confirmed(), []string{"u2"}; !slices.Equal(got, want) {
		t.Errorf("Confirmed() = %v, want %v", got, want)
	}
	if got, want := ev.Waitlisted(), []string{"u3"}; !slices.Equal(got, want) {
		t.Errorf("Waitlisted() = %v, want %v", got, want)
	}
	if got, want := n.Notified(), []string{"u2"}; !slices.Equal(got, want) {
		t.Errorf("Notified() = %v, want %v", got, want)
	}
}

// ANCHOR_END: r1

// ANCHOR: r2

// R2：額滿時報名，呼叫端要分得出「報名成功」與「排進候補」。
func TestRegisterReportsWaitlisted_R2(t *testing.T) {
	ev := newEvent(1, &fakeNotifier{})
	if err := ev.Register("u1"); err != nil {
		t.Fatalf("Register(u1) = %v, want nil", err)
	}
	if err := ev.Register("u2"); !errors.Is(err, signup.ErrWaitlisted) {
		t.Errorf("Register(u2) when full = %v, want ErrWaitlisted", err)
	}
}

// ANCHOR_END: r2

// blockingNotifier 的 Notify 先送出 entered，再一直卡到 release 關閉，模擬很慢的通知服務。
type blockingNotifier struct {
	entered chan<- struct{}
	release <-chan struct{}
}

// Notify 實作 signup.Notifier：告知測試已經進來，然後卡住。
func (b blockingNotifier) Notify(string) {
	b.entered <- struct{}{}
	<-b.release
}

// ANCHOR: r3

// R3：通知卡住時，其他報名仍然要完成。
// 測試假設取消會遞補並通知（R1 已修）；通知卡在 release 上，另一個 goroutine 的報名不能被擋住。
// 全部在 synctest 的 bubble 裡用 channel 同步，沒有 sleep；Register 若被鎖擋住，
// bubble 無法推進，測試會卡到逾時。
func TestSlowNotifierDoesNotBlockRegister_R3(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		entered, release := make(chan struct{}), make(chan struct{})
		ev := newEvent(1, blockingNotifier{entered: entered, release: release})
		join(t, ev, "u1")
		join(t, ev, "u2")

		cancelled := make(chan error)
		go func() { cancelled <- ev.Cancel("u1") }()
		<-entered // u2 已遞補，通知卡在 release 上

		registered := make(chan error)
		go func() { registered <- ev.Register("u3") }()
		if err := <-registered; !errors.Is(err, signup.ErrWaitlisted) {
			t.Errorf("Register(u3) while notifying = %v, want ErrWaitlisted", err)
		}

		close(release)
		if err := <-cancelled; err != nil {
			t.Errorf("Cancel(u1) = %v, want nil", err)
		}
	})
}

// ANCHOR_END: r3

// ANCHOR: r4

// R4：已經在候補名單裡的人再報名，要被拒絕，不能重複排入。
func TestRegisterWaitlistedTwice_R4(t *testing.T) {
	ev := newEvent(1, &fakeNotifier{})
	join(t, ev, "u1")
	join(t, ev, "u2")

	if err := ev.Register("u2"); !errors.Is(err, signup.ErrAlreadyRegistered) {
		t.Errorf("second Register(u2) = %v, want ErrAlreadyRegistered", err)
	}
	if got, want := ev.Waitlisted(), []string{"u2"}; !slices.Equal(got, want) {
		t.Errorf("Waitlisted() = %v, want %v", got, want)
	}
}

// ANCHOR_END: r4

// ANCHOR: r6

// R6：取消的是候補者本人時，只從候補名單移除，不遞補、不通知。
func TestCancelByWaitlistedPerson_R6(t *testing.T) {
	n := &fakeNotifier{}
	ev := newEvent(1, n)
	join(t, ev, "u1")
	join(t, ev, "u2")
	join(t, ev, "u3")

	if err := ev.Cancel("u2"); err != nil {
		t.Fatal(err)
	}

	if got, want := ev.Confirmed(), []string{"u1"}; !slices.Equal(got, want) {
		t.Errorf("Confirmed() = %v, want %v", got, want)
	}
	if got, want := ev.Waitlisted(), []string{"u3"}; !slices.Equal(got, want) {
		t.Errorf("Waitlisted() = %v, want %v", got, want)
	}
	if got := n.Notified(); len(got) != 0 {
		t.Errorf("Notified() = %v, want none", got)
	}
}

// ANCHOR_END: r6
