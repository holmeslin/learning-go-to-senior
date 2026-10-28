package loyalty_test

import (
	"testing"

	loyalty "github.com/holmeslin/learning-go-to-senior/examples/chapter3/03_redundant_interface/before"
)

// ANCHOR: fake

// fakeStore 是 Store 的手寫替身：用一張 map 記點數。
type fakeStore struct {
	points map[string]int
}

func (f *fakeStore) Get(id string) (loyalty.Member, error) {
	return loyalty.Member{ID: id, Points: f.points[id]}, nil
}

func (f *fakeStore) Add(id string, n int) (loyalty.Member, error) {
	f.points[id] += n
	return loyalty.Member{ID: id, Points: f.points[id]}, nil
}

// ANCHOR_END: fake

// fakeNotifier 記下收到通知的會員。
type fakeNotifier struct {
	sent []string
}

func (f *fakeNotifier) Notify(id, msg string) error {
	f.sent = append(f.sent, id)
	return nil
}

func TestEarnNotifiesOnceAtGold(t *testing.T) {
	n := &fakeNotifier{}
	svc := loyalty.NewService(&fakeStore{points: map[string]int{"alice": 95}}, n)

	for range 2 {
		if _, err := svc.Earn("alice", 100); err != nil {
			t.Fatal(err)
		}
	}

	if len(n.sent) != 1 {
		t.Errorf("notified %v, want exactly once", n.sent)
	}
}
