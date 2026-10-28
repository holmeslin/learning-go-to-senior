package notify_test

import (
	"context"
	"errors"
	"slices"
	"sync/atomic"
	"testing"
	"testing/synctest"

	notify "github.com/holmeslin/learning-go-to-senior/examples/chapter6/01_goroutine_ownership/after"
)

// isClosed 回報 ch 是否已經關閉，不會阻塞。
func isClosed(ch <-chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}

// ANCHOR: broadcast

func TestBroadcastReturnsAfterEverySend(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		release := make(chan struct{})
		var sent atomic.Int32
		n := notify.New(func(string) error {
			<-release
			sent.Add(1)
			return nil
		})

		returned := make(chan struct{})
		go func() {
			n.Broadcast([]string{"a@example.com", "b@example.com", "c@example.com"})
			close(returned)
		}()

		synctest.Wait() // 三封信都卡在 release，Broadcast 只能在等它們
		if isClosed(returned) {
			t.Error("Broadcast returned while sends were still running")
		}
		close(release)
		synctest.Wait()
		if !isClosed(returned) {
			t.Fatal("Broadcast did not return after every send finished")
		}
		if got, want := sent.Load(), int32(3); got != want {
			t.Errorf("sent = %d, want %d", got, want)
		}
	})
}

// ANCHOR_END: broadcast

func TestBroadcastReturnsFailedInInputOrder(t *testing.T) {
	bad := map[string]bool{"x@example.com": true, "z@example.com": true}
	n := notify.New(func(to string) error {
		if bad[to] {
			return errors.New("mailbox full")
		}
		return nil
	})

	got := n.Broadcast([]string{"x@example.com", "y@example.com", "z@example.com"})
	if want := []string{"x@example.com", "z@example.com"}; !slices.Equal(got, want) {
		t.Errorf("Broadcast() = %v, want %v", got, want)
	}
}

// ANCHOR: run

func TestRunReturnsAfterInFlightSend(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		release := make(chan struct{})
		var sent []string // 只有 Run 的 goroutine 會寫，Run 回傳後才讀
		n := notify.New(func(to string) error {
			<-release
			sent = append(sent, to)
			return nil
		})
		ctx, stop := context.WithCancel(t.Context())
		stopped := make(chan struct{})
		go func() {
			n.Run(ctx)
			close(stopped)
		}()
		n.Welcome("a@example.com")
		n.Welcome("b@example.com")

		synctest.Wait() // 第一封卡在 release
		stop()
		synctest.Wait()
		if isClosed(stopped) {
			t.Error("Run returned while a send was still running")
		}
		close(release)
		synctest.Wait()
		if !isClosed(stopped) {
			t.Fatal("Run did not return after the in-flight send finished")
		}
		if want := []string{"a@example.com"}; !slices.Equal(sent, want) {
			t.Errorf("sent = %v, want %v (queued mail is not sent after stop)", sent, want)
		}
	})
}

// ANCHOR_END: run
