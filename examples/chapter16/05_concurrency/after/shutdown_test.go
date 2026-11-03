package shutdown_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter16/05_concurrency/after"
)

// fakeSender 是假的推播：每個請求占用 delay 的假時間；hang 為 true 時不回應，直到請求的 ctx 結束。
type fakeSender struct {
	delay time.Duration
	hang  bool

	mu    sync.Mutex
	calls int // 收到的請求數
	cut   int // 被 ctx 切斷的請求數
}

func (f *fakeSender) Send(ctx context.Context, _ string) error {
	f.mu.Lock()
	f.calls++
	f.mu.Unlock()

	var reply <-chan time.Time // nil：永遠不回應
	if !f.hang {
		reply = time.After(f.delay)
	}
	select {
	case <-reply:
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

// start 在背景啟動服務。cancel 代表「收到結束訊號」；Run 的結果從 done 讀。
func start(t *testing.T, svc *shutdown.Service, due []string) (cancel context.CancelFunc, done <-chan error) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	result := make(chan error, 1)
	go func() { result <- svc.Run(ctx, due) }()
	return cancel, result
}

// ANCHOR: finish

// 收到結束訊號之後：進行中的送出做完、記為已寄出，Run 在做完之後才回傳；還沒開始的提醒沒有紀錄。
// 「沒有殘留的 goroutine」由 synctest.Test 保證：bubble 裡還有 goroutine 沒結束，測試就失敗。
func TestRunLetsInFlightSendsFinish(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		sender := &fakeSender{delay: 3 * time.Second}
		svc := &shutdown.Service{
			Ledger: shutdown.NewLedger(), Sender: sender,
			Concurrency: 2, RequestTimeout: 5 * time.Second, ShutdownTimeout: 30 * time.Second,
		}
		due := keys(5)
		cancel, done := start(t, svc, due)

		time.Sleep(time.Second) // 兩個請求在路上，各要三秒
		cancelledAt := time.Now()
		cancel()
		err := <-done

		if err != nil {
			t.Errorf("Run() error = %v, want nil", err)
		}
		// 請求在 0 秒出發、3 秒做完；訊號在第 1 秒來，所以 Run 在訊號之後 2 秒回傳。
		if got, want := time.Since(cancelledAt), 2*time.Second; got != want {
			t.Errorf("Run() returned %v after the signal, want %v: the in-flight sends", got, want)
		}
		if calls, cut := sender.counts(); calls != 2 || cut != 0 {
			t.Errorf("sends = %d, cut = %d, want 2 sends and none cut", calls, cut)
		}
		for _, key := range due[:2] {
			if got, _ := svc.Ledger.State(key); got != shutdown.StateSent {
				t.Errorf("in-flight %s: state = %q, want sent", key, got)
			}
		}
		for _, key := range due[2:] {
			if got, ok := svc.Ledger.State(key); ok {
				t.Errorf("%s never started: state = %q, want no record", key, got)
			}
		}
	})
}

// ANCHOR_END: finish

// ANCHOR: deadline

// 供應商不回應時，等的時間受每個請求的逾時限制：關機最多等一個 RequestTimeout，遠低於 ShutdownTimeout；
// 送出記為待重試，紀錄不會停在寄送中。
func TestRunWaitsAtMostOneRequestTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		sender := &fakeSender{hang: true}
		svc := &shutdown.Service{
			Ledger: shutdown.NewLedger(), Sender: sender,
			Concurrency: 2, RequestTimeout: 5 * time.Second, ShutdownTimeout: 30 * time.Second,
		}
		due := keys(2)
		cancel, done := start(t, svc, due)

		synctest.Wait() // 兩個請求都在路上
		cancelledAt := time.Now()
		cancel()
		err := <-done

		if err != nil {
			t.Errorf("Run() error = %v, want nil", err)
		}
		if got, want := time.Since(cancelledAt), 5*time.Second; got != want {
			t.Errorf("Run() returned %v after the signal, want %v: one request timeout", got, want)
		}
		for _, key := range due {
			if got, _ := svc.Ledger.State(key); got != shutdown.StateRetrying {
				t.Errorf("%s: state = %q, want retrying", key, got)
			}
		}
	})
}

// 關機期限比請求的逾時短（設定檢查本來會擋下這樣的組合）：期限一到，Run 就放棄等待、回傳錯誤，
// 紀錄簿也關了，停在寄送中的紀錄只能留給之後逾期的檢查處理。
func TestRunReportsAnErrorWhenTheDeadlinePasses(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		sender := &fakeSender{hang: true}
		svc := &shutdown.Service{
			Ledger: shutdown.NewLedger(), Sender: sender,
			Concurrency: 1, RequestTimeout: 30 * time.Second, ShutdownTimeout: 2 * time.Second,
		}
		due := keys(1)
		cancel, done := start(t, svc, due)

		synctest.Wait()
		cancelledAt := time.Now()
		cancel()
		err := <-done

		if !errors.Is(err, context.DeadlineExceeded) || time.Since(cancelledAt) != 2*time.Second {
			t.Errorf("Run() = %v after %v, want DeadlineExceeded after 2s", err, time.Since(cancelledAt))
		}
		if got, _ := svc.Ledger.State(due[0]); got != shutdown.StateSending {
			t.Errorf("%s: state = %q, want it stuck in sending", due[0], got)
		}
		time.Sleep(time.Minute) // 殘留的請求在自己的逾時之後結束，bubble 才收得掉
	})
}

// ANCHOR_END: deadline
