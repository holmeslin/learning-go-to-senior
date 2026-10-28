package signup_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter4/07_fake_or_real_database/after/signup"
)

// failingStore 讓 Add 永遠失敗，用來測 Invite 遇到其他錯誤時的行為。
type failingStore struct{ err error }

func (f failingStore) Add(context.Context, string) (signup.User, error) {
	return signup.User{}, f.err
}

func TestInviteAddsEveryone(t *testing.T) {
	t.Parallel()
	svc := signup.NewService(newFakeStore())

	added, skipped, err := svc.Invite(t.Context(), []string{"ann@example.com", "bob@example.com"})

	if err != nil {
		t.Fatalf("Invite() error = %v", err)
	}
	if added != 2 || len(skipped) != 0 {
		t.Errorf("Invite() = %d added, %v skipped, want 2 added, none skipped", added, skipped)
	}
}

// ANCHOR: test

func TestInviteSkipsTakenEmails(t *testing.T) {
	t.Parallel()
	svc := signup.NewService(newFakeStore())

	added, skipped, err := svc.Invite(t.Context(), []string{"ann@example.com", "bob@example.com", "ann@example.com"})

	if err != nil {
		t.Fatalf("Invite() error = %v", err)
	}
	if want := []string{"ann@example.com"}; added != 2 || !slices.Equal(skipped, want) {
		t.Errorf("Invite() = %d added, %v skipped, want 2 added, %v skipped", added, skipped, want)
	}
}

// ANCHOR_END: test

func TestInviteStopsOnOtherErrors(t *testing.T) {
	t.Parallel()
	errDown := errors.New("database down")
	svc := signup.NewService(failingStore{err: errDown})

	added, _, err := svc.Invite(t.Context(), []string{"ann@example.com", "bob@example.com"})

	if !errors.Is(err, errDown) {
		t.Errorf("Invite() error = %v, want it to wrap %v", err, errDown)
	}
	if added != 0 {
		t.Errorf("Invite() added %d, want 0", added)
	}
}
