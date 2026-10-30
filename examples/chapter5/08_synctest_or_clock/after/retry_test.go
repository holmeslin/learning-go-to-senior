package retry_test

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"testing"
	"testing/synctest"
	"time"

	retry "github.com/holmeslin/learning-go-to-senior/examples/chapter5/08_synctest_or_clock/after"
)

// ANCHOR: backoff

func TestDoBackoff(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		calls := 0
		fn := func(context.Context) error {
			calls++
			if calls < 3 {
				return errors.New("not ready")
			}
			return nil
		}
		p := retry.Policy{Attempts: 3, Timeout: 30 * time.Second, Backoff: time.Second}

		start := time.Now()
		done := make(chan error)
		go func() { done <- retry.Do(t.Context(), p, fn) }()

		synctest.Sleep(time.Second) // 1 秒後：第二次剛失敗，正在等 2 秒
		if calls != 2 {
			t.Fatalf("calls after 1s = %d, want 2", calls)
		}
		if err := <-done; err != nil {
			t.Fatalf("Do() = %v, want nil", err)
		}
		if got, want := time.Since(start), 3*time.Second; got != want {
			t.Errorf("Do() took %v, want %v", got, want)
		}
	})
}

// ANCHOR_END: backoff

// ANCHOR: timeout

func TestDoAttemptTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		calls := 0
		hang := func(ctx context.Context) error {
			calls++
			<-ctx.Done()
			return ctx.Err()
		}
		p := retry.Policy{Attempts: 3, Timeout: 30 * time.Second, Backoff: time.Second}

		start := time.Now()
		err := retry.Do(t.Context(), p, hang)

		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("Do() = %v, want context.DeadlineExceeded", err)
		}
		if calls != 3 {
			t.Errorf("calls = %d, want 3", calls)
		}
		// 30 + 1 + 30 + 2 + 30 秒，在 bubble 裡是精確的。
		if got, want := time.Since(start), 93*time.Second; got != want {
			t.Errorf("Do() took %v, want %v", got, want)
		}
	})
}

// ANCHOR_END: timeout

// ANCHOR: cancel

func TestDoStopsWhenContextEnds(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
		defer cancel()
		fail := func(context.Context) error { return errors.New("not ready") }
		p := retry.Policy{Attempts: 3, Timeout: 30 * time.Second, Backoff: time.Minute}

		start := time.Now()
		err := retry.Do(ctx, p, fail)

		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("Do() = %v, want context.DeadlineExceeded", err)
		}
		if got, want := time.Since(start), 10*time.Second; got != want {
			t.Errorf("Do() returned after %v, want %v", got, want)
		}
	})
}

// ANCHOR_END: cancel

// ANCHOR: pipe

func TestDoOverPipe(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		client, server := net.Pipe() // 行程內的假連線，不經過 socket
		defer client.Close()
		defer server.Close()
		go func() { _, _ = io.Copy(io.Discard, server) }() // 對方收下請求，但永遠不回

		ask := func(ctx context.Context) error {
			deadline, _ := ctx.Deadline()
			_ = client.SetDeadline(deadline)
			if _, err := client.Write([]byte("ping")); err != nil {
				return err
			}
			_, err := client.Read(make([]byte, 4))
			return err
		}
		p := retry.Policy{Attempts: 2, Timeout: 5 * time.Second, Backoff: time.Second}

		start := time.Now()
		err := retry.Do(t.Context(), p, ask)

		if !errors.Is(err, os.ErrDeadlineExceeded) {
			t.Errorf("Do() = %v, want os.ErrDeadlineExceeded", err)
		}
		if got, want := time.Since(start), 11*time.Second; got != want {
			t.Errorf("Do() took %v, want %v", got, want)
		}
	})
}

// ANCHOR_END: pipe
