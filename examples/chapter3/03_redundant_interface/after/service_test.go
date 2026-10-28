package loyalty_test

import (
	"errors"
	"testing"

	loyalty "github.com/holmeslin/learning-go-to-senior/examples/chapter3/03_redundant_interface/after"
)

// ANCHOR: fake

// fakeNotifier 記下收到通知的會員；err 不是 nil 時，Notify 會失敗。
type fakeNotifier struct {
	sent []string
	err  error
}

func (f *fakeNotifier) Notify(id, msg string) error {
	if f.err != nil {
		return f.err
	}
	f.sent = append(f.sent, id)
	return nil
}

// ANCHOR_END: fake

// ANCHOR: test

func TestEarnNotifiesOnceAtGold(t *testing.T) {
	n := &fakeNotifier{}
	store := loyalty.NewStore(loyalty.Member{ID: "alice", Points: 95})
	svc := loyalty.NewService(store, n)

	for range 2 {
		if _, err := svc.Earn("alice", 100); err != nil {
			t.Fatal(err)
		}
	}

	if len(n.sent) != 1 || n.sent[0] != "alice" {
		t.Errorf("notified %v, want [alice] exactly once", n.sent)
	}
	if got, _ := svc.Balance("alice"); got != 115 {
		t.Errorf("balance = %d, want 115", got)
	}
}

// ANCHOR_END: test

func TestEarnPoints(t *testing.T) {
	tests := []struct {
		name  string
		spent int
		want  int
	}{
		{"每 10 元 1 點", 250, 25},
		{"不足 10 元不得點", 9, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := loyalty.NewService(loyalty.NewStore(loyalty.Member{ID: "alice"}), &fakeNotifier{})
			got, err := svc.Earn("alice", tt.spent)
			if err != nil || got != tt.want {
				t.Errorf("Earn(%d) = %d, %v; want %d, nil", tt.spent, got, err, tt.want)
			}
		})
	}
}

func TestEarnRejectsNonPositiveSpent(t *testing.T) {
	svc := loyalty.NewService(loyalty.NewStore(loyalty.Member{ID: "alice"}), &fakeNotifier{})
	for _, spent := range []int{0, -100} {
		if _, err := svc.Earn("alice", spent); err == nil {
			t.Errorf("Earn(%d) error = nil, want error", spent)
		}
	}
	if got, _ := svc.Balance("alice"); got != 0 {
		t.Errorf("balance = %d, want 0", got)
	}
}

func TestEarnUnknownMember(t *testing.T) {
	n := &fakeNotifier{}
	svc := loyalty.NewService(loyalty.NewStore(), n)

	_, err := svc.Earn("ghost", 200)

	if !errors.Is(err, loyalty.ErrNotFound) {
		t.Errorf("error = %v, want ErrNotFound", err)
	}
	if len(n.sent) != 0 {
		t.Errorf("notified %v, want nobody", n.sent)
	}
}

func TestEarnNotifyFailureKeepsPoints(t *testing.T) {
	errDown := errors.New("webhook down")
	n := &fakeNotifier{err: errDown}
	svc := loyalty.NewService(loyalty.NewStore(loyalty.Member{ID: "alice", Points: 95}), n)

	got, err := svc.Earn("alice", 100)

	if !errors.Is(err, errDown) {
		t.Fatalf("error = %v, want it to wrap errDown", err)
	}
	if got != 105 {
		t.Errorf("Earn returned %d, want 105", got)
	}
	// 通知失敗不代表點數沒入帳。
	if bal, _ := svc.Balance("alice"); bal != 105 {
		t.Errorf("balance = %d, want 105", bal)
	}
}

func TestBalanceUnknownMember(t *testing.T) {
	svc := loyalty.NewService(loyalty.NewStore(), &fakeNotifier{})
	if _, err := svc.Balance("ghost"); !errors.Is(err, loyalty.ErrNotFound) {
		t.Errorf("error = %v, want ErrNotFound", err)
	}
}
