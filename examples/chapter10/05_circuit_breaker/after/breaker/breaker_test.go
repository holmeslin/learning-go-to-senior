package breaker_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter10/05_circuit_breaker/after/breaker"
)

// ANCHOR: helpers

// clock 是測試用的時鐘：時間只在 Advance 時前進。
type clock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *clock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

var errBoom = errors.New("boom")

// newBreaker 回傳連續失敗 3 次就打開、冷卻 10 秒的斷路器，與它的時鐘。
func newBreaker() (*breaker.Breaker, *clock) {
	clk := &clock{t: time.Unix(0, 0)}
	return breaker.New(breaker.Config{Threshold: 3, Cooldown: 10 * time.Second, Now: clk.Now}), clk
}

// ANCHOR_END: helpers

// ANCHOR: transitions

func TestStateTransitions(t *testing.T) {
	b, clk := newBreaker()
	calls := 0
	call := func(err error) error { return b.Do(func() error { calls++; return err }) }
	wantState := func(want breaker.State) {
		t.Helper()
		if got := b.State(); got != want {
			t.Fatalf("state = %v, want %v", got, want)
		}
	}

	// 連續失敗才算數：中間成功一次，計數歸零。
	_ = call(errBoom)
	_ = call(errBoom)
	_ = call(nil)
	_ = call(errBoom)
	_ = call(errBoom)
	wantState(breaker.Closed)
	_ = call(errBoom) // 連續第 3 次
	wantState(breaker.Open)

	// 打開時：立刻回 ErrOpen，不呼叫下游。
	before := calls
	if err := call(nil); !errors.Is(err, breaker.ErrOpen) || calls != before {
		t.Fatalf("while open: err = %v, calls %d -> %d; want ErrOpen and no call", err, before, calls)
	}
	clk.Advance(9 * time.Second)
	if err := call(nil); !errors.Is(err, breaker.ErrOpen) {
		t.Fatalf("1s before the cooldown ends: err = %v, want ErrOpen", err)
	}

	// 冷卻結束：下一個請求是試探；試探失敗，重新打開並重新計算冷卻。
	clk.Advance(time.Second)
	err := b.Do(func() error {
		wantState(breaker.HalfOpen)
		return errBoom
	})
	if !errors.Is(err, errBoom) {
		t.Fatalf("probe err = %v, want the downstream error", err)
	}
	wantState(breaker.Open)
	clk.Advance(9 * time.Second)
	if err := call(nil); !errors.Is(err, breaker.ErrOpen) {
		t.Fatalf("cooldown restarts after a failed probe: err = %v, want ErrOpen", err)
	}

	// 這次試探成功，才關上；失敗計數從零開始。
	clk.Advance(time.Second)
	if err := call(nil); err != nil {
		t.Fatalf("probe err = %v, want nil", err)
	}
	wantState(breaker.Closed)
	_ = call(errBoom)
	_ = call(errBoom)
	wantState(breaker.Closed)
}

// ANCHOR_END: transitions

// ANCHOR: probe

func TestHalfOpenAdmitsOneProbe(t *testing.T) {
	b, clk := newBreaker()
	for range 3 {
		_ = b.Do(func() error { return errBoom })
	}
	clk.Advance(10 * time.Second)

	entered, release := make(chan struct{}), make(chan struct{})
	probeDone := make(chan error)
	go func() {
		probeDone <- b.Do(func() error {
			close(entered)
			<-release // 試探還在途中
			return nil
		})
	}()
	<-entered

	for range 5 {
		err := b.Do(func() error {
			t.Error("a second request reached the downstream while the probe was in flight")
			return nil
		})
		if !errors.Is(err, breaker.ErrOpen) {
			t.Errorf("request during the probe: err = %v, want ErrOpen", err)
		}
	}
	if got := b.State(); got != breaker.HalfOpen {
		t.Errorf("state during the probe = %v, want half-open", got)
	}

	close(release)
	if err := <-probeDone; err != nil {
		t.Errorf("probe err = %v, want nil", err)
	}
	if got := b.State(); got != breaker.Closed {
		t.Errorf("state after the probe succeeded = %v, want closed", got)
	}
}

// ANCHOR_END: probe

func TestLateFailureDoesNotExtendCooldown(t *testing.T) {
	b, clk := newBreaker()
	entered, release := make(chan struct{}), make(chan struct{})
	slowDone := make(chan error)
	go func() {
		slowDone <- b.Do(func() error { close(entered); <-release; return errBoom })
	}()
	<-entered // 這個請求在斷路器還關著時就出發了
	for range 3 {
		_ = b.Do(func() error { return errBoom })
	}
	clk.Advance(9 * time.Second)

	close(release)
	if err := <-slowDone; !errors.Is(err, errBoom) { // 打開之前出發的請求，這時才失敗
		t.Fatalf("late request err = %v, want errBoom", err)
	}
	clk.Advance(time.Second)
	if err := b.Do(func() error { return nil }); err != nil {
		t.Errorf("err = %v, want nil: a late failure must not restart the cooldown", err)
	}
}

func TestPanicInProbeDoesNotStickHalfOpen(t *testing.T) {
	b, clk := newBreaker()
	for range 3 {
		_ = b.Do(func() error { return errBoom })
	}
	clk.Advance(10 * time.Second)

	func() {
		defer func() { _ = recover() }()
		_ = b.Do(func() error { panic("downstream client bug") })
	}()
	if got := b.State(); got != breaker.Open {
		t.Fatalf("state after a panicking probe = %v, want open", got)
	}
	clk.Advance(10 * time.Second)
	if err := b.Do(func() error { return nil }); err != nil {
		t.Errorf("next probe err = %v, want nil", err)
	}
}

func TestNewDefaults(t *testing.T) {
	b := breaker.New(breaker.Config{}) // Threshold 為 0 當作 1，Now 為 nil 用 time.Now
	_ = b.Do(func() error { return errBoom })
	if got := b.State(); got != breaker.Open {
		t.Errorf("state after 1 failure with default threshold = %v, want open", got)
	}
}
