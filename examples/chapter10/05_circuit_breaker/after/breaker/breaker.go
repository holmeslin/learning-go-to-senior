// Package breaker 是一個簡單的斷路器：下游連續失敗就打開，冷卻之後放一個試探請求。
package breaker

import (
	"errors"
	"sync"
	"time"
)

// ErrOpen 表示斷路器擋下了這個請求，沒有呼叫下游。
var ErrOpen = errors.New("breaker: open")

// ANCHOR: types

// State 是斷路器的狀態。匯出它是為了讓測試能斷言狀態轉換；
// 正式環境也會把它放進指標與日誌。
type State int

const (
	Closed   State = iota // 關閉：請求正常通過
	Open                  // 打開：請求立刻被擋下
	HalfOpen              // 半開：只放行一個新的試探請求，其餘新請求被擋下
)

// String 回傳狀態的名稱，給日誌與指標用。
func (s State) String() string {
	return [...]string{"closed", "open", "half-open"}[s]
}

// Config 是斷路器的設定。
type Config struct {
	Threshold int              // 連續失敗幾次就打開；小於 1 當作 1
	Cooldown  time.Duration    // 打開之後多久放一個試探請求
	Now       func() time.Time // 注入的時鐘；nil 用 time.Now
}

// Breaker 是一個下游一個的斷路器，可以同時被多個 goroutine 使用。
type Breaker struct {
	cfg Config

	mu       sync.Mutex
	state    State
	failures int       // Closed 時的連續失敗次數
	openedAt time.Time // 最近一次打開的時間
}

// New 依 cfg 建立一個關閉的斷路器。
func New(cfg Config) *Breaker {
	cfg.Threshold = max(cfg.Threshold, 1)
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &Breaker{cfg: cfg}
}

// State 回傳目前的狀態。
func (b *Breaker) State() State {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.state
}

// ANCHOR_END: types

// ANCHOR: do

// Do 在斷路器允許時執行 fn。被擋下時回傳 ErrOpen，不呼叫 fn；
// 否則回傳 fn 的錯誤，並依結果更新狀態。fn 回傳的每個非 nil 錯誤都算失敗。
func (b *Breaker) Do(fn func() error) (err error) {
	probe, err := b.admit()
	if err != nil {
		return err
	}
	ok := false
	defer func() { b.record(probe, ok) }() // fn panic 時 ok 仍是 false
	err = fn()
	ok = err == nil
	return err
}

// admit 決定這個請求能不能過；probe 為 true 代表它就是半開時的那個試探。
func (b *Breaker) admit() (probe bool, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	switch b.state {
	case Open:
		if b.cfg.Now().Sub(b.openedAt) < b.cfg.Cooldown {
			return false, ErrOpen
		}
		b.state = HalfOpen // 冷卻結束：第一個到的請求當試探
		return true, nil
	case HalfOpen:
		return false, ErrOpen // 試探還在途中
	}
	return false, nil
}

func (b *Breaker) record(probe, ok bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	switch {
	case probe && ok:
		b.state, b.failures = Closed, 0
	case probe:
		b.state, b.openedAt = Open, b.cfg.Now()
	case b.state != Closed:
		// 斷路器打開之前就出發的請求，晚到的結果不算數。
	case ok:
		b.failures = 0
	default:
		b.failures++
		if b.failures >= b.cfg.Threshold {
			b.state, b.openedAt = Open, b.cfg.Now()
		}
	}
}

// ANCHOR_END: do
