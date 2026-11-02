package signup_test

import (
	"net/http"
	"slices"
	"sync"
	"testing"

	signup "github.com/holmeslin/learning-go-to-senior/examples/chapter13/01_design_before_details/before"
)

// fakeNotifier 記錄被通知的人，供測試斷言。
type fakeNotifier struct {
	mu  sync.Mutex
	who []string
}

// Notify 實作 signup.Notifier：記下 who。
func (f *fakeNotifier) Notify(who string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.who = append(f.who, who)
}

// Notified 回傳到目前為止被通知的人，依通知順序。
func (f *fakeNotifier) Notified() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.who)
}

func TestWaitlistPromotesInOrder(t *testing.T) {
	n := &fakeNotifier{}
	ev := signup.NewEvent(1, n)
	for _, who := range []string{"u1", "u2", "u3"} {
		if err := ev.Register(who); err != nil {
			t.Fatalf("Register(%q) = %v, want nil", who, err)
		}
	}
	if got, want := ev.Waitlisted(), []string{"u2", "u3"}; !slices.Equal(got, want) {
		t.Fatalf("Waitlisted() = %v, want %v", got, want)
	}

	if err := ev.Cancel("u1"); err != nil {
		t.Fatal(err)
	}
	if who, ok := ev.PromoteNext(); !ok || who != "u2" {
		t.Errorf("PromoteNext() = %q, %v, want u2, true", who, ok)
	}
	if got, want := ev.Confirmed(), []string{"u2"}; !slices.Equal(got, want) {
		t.Errorf("Confirmed() = %v, want %v", got, want)
	}
	if got, want := n.Notified(), []string{"u2"}; !slices.Equal(got, want) {
		t.Errorf("Notified() = %v, want %v", got, want)
	}
}

func TestHandlerCancelPromotesWaitlisted(t *testing.T) {
	n := &fakeNotifier{}
	store := signup.NewStore(n)
	if _, err := store.Create("go-meetup", 1); err != nil {
		t.Fatal(err)
	}
	h := signup.NewHandler(store)

	do(t, h, "POST", "/events/go-meetup/registrations", `{"who":"u1"}`)
	do(t, h, "POST", "/events/go-meetup/registrations", `{"who":"u2"}`)
	if code, _ := do(t, h, "DELETE", "/events/go-meetup/registrations/u1", ""); code != http.StatusNoContent {
		t.Fatalf("cancel = %d, want 204", code)
	}

	ev, _ := store.Get("go-meetup")
	if got, want := ev.Confirmed(), []string{"u2"}; !slices.Equal(got, want) {
		t.Errorf("Confirmed() = %v, want %v", got, want)
	}
	if got, want := n.Notified(), []string{"u2"}; !slices.Equal(got, want) {
		t.Errorf("Notified() = %v, want %v", got, want)
	}
}
