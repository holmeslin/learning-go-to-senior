package retry_test

import (
	"context"
	"errors"
	"testing"
	"time"

	retry "github.com/holmeslin/learning-go-to-senior/examples/chapter5/08_synctest_or_clock/before"
)

// ANCHOR: test

func TestDoSucceedsFirstTry(t *testing.T) {
	calls := 0
	fn := func(context.Context) error {
		calls++
		return nil
	}
	p := retry.Policy{Attempts: 3, Timeout: 30 * time.Second, Backoff: time.Second}

	if err := retry.Do(context.Background(), p, fn); err != nil {
		t.Fatalf("Do() = %v, want nil", err)
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1", calls)
	}
}

func TestDoSingleAttemptFails(t *testing.T) {
	boom := errors.New("boom")
	fn := func(context.Context) error { return boom }
	p := retry.Policy{Attempts: 1, Timeout: 30 * time.Second}

	if err := retry.Do(context.Background(), p, fn); !errors.Is(err, boom) {
		t.Errorf("Do() = %v, want error wrapping boom", err)
	}
}

// ANCHOR_END: test
