package shutdown_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter16/05_concurrency/before"
)

// fakeSender 是假的推播：每個請求占用 delay 的假時間，請求的 ctx 結束就放棄。
type fakeSender struct {
	delay time.Duration

	mu    sync.Mutex
	calls int // 收到的請求數
	cut   int // 被 ctx 切斷的請求數
}

func (f *fakeSender) Send(ctx context.Context, _ string) error {
	f.mu.Lock()
	f.calls++
	f.mu.Unlock()

	select {
	case <-time.After(f.delay):
		return nil
	case <-ctx.Done():
		f.mu.Lock()
		f.cut++
		f.mu.Unlock()
		return ctx.Err()
	}
}

func (f *fakeSender) counts() (calls, cut int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls, f.cut
}

// keys 回傳 n 則提醒的冪等鍵。
func keys(n int) []string {
	due := make([]string, n)
	for i := range due {
		due[i] = fmt.Sprintf("reminder-1-%d", i+1)
	}
	return due
}

// ANCHOR: cut

// 這個測試鎖住問題：收到結束訊號，Run 立刻回傳；進行中的兩個送出被切斷，
// 紀錄停在寄送中，結果一個也沒寫進去。
func TestRunCutsInFlightSends(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		sender := &fakeSender{delay: 3 * time.Second}
		svc := &shutdown.Service{Ledger: shutdown.NewLedger(), Sender: sender, Concurrency: 2}
		due := keys(5)
		ctx, cancel := context.WithCancel(t.Context())
		done := make(chan error, 1)
		go func() { done <- svc.Run(ctx, due) }()

		time.Sleep(time.Second) // 兩個請求在路上，各要三秒
		cancelledAt := time.Now()
		cancel()
		<-done
		if got := time.Since(cancelledAt); got != 0 {
			t.Errorf("Run() returned %v after the signal, want 0: nobody waits for the sends", got)
		}

		synctest.Wait() // 被切斷的 goroutine 收尾
		if calls, cut := sender.counts(); calls != 2 || cut != 2 {
			t.Errorf("sends = %d, cut = %d, want both of the 2 sends cut", calls, cut)
		}
		for _, key := range due[:2] {
			if got, _ := svc.Ledger.State(key); got != shutdown.StateSending {
				t.Errorf("%s: state = %q, want it stuck in sending", key, got)
			}
		}
	})
}

// ANCHOR_END: cut
