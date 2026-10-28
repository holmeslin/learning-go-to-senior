package signup_test

import (
	"testing"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter4/07_fake_or_real_database/before/signup"
)

// ANCHOR: test

func TestInviteAddsEveryone(t *testing.T) {
	svc := signup.NewService(newFakeStore())

	added, skipped, err := svc.Invite(t.Context(), []string{"ann@example.com", "bob@example.com"})

	if err != nil {
		t.Fatalf("Invite() error = %v", err)
	}
	if added != 2 || len(skipped) != 0 {
		t.Errorf("Invite() = %d added, %v skipped, want 2 added, none skipped", added, skipped)
	}
}

// ANCHOR_END: test
