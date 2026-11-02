// Package shutdown 是提醒服務關機順序的縮小版，這個目錄是常見寫法：
// 收到結束訊號就關紀錄簿、回傳，沒有人等進行中的送出。
//
// 刻意保留的問題（lint 已排除 before/）：goroutine 開了就不管；結束訊號一路傳進送出與寫入，
// 切斷進行中的送出，結果也寫不進去；先關紀錄簿，之後的寫入都失敗。較好的做法在 ../after。
// Ledger 代替寄送紀錄表，Service.Run 代替服務本體的 Service.Run；不 import 服務本體。
package shutdown

import (
	"context"
	"errors"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

// State 是一則提醒在寄送紀錄裡的狀態。
type State string

const (
	// StateSending 是已寫紀錄、還沒確認結果。
	StateSending State = "sending"
	// StateSent 是已寄出。
	StateSent State = "sent"
	// StateRetrying 是送出失敗，等下次掃描重試。
	StateRetrying State = "retrying"
)

// ErrClosed 是紀錄簿關閉之後，寫入回傳的錯誤；對應 database/sql 的 "sql: database is closed"。
var ErrClosed = errors.New("shutdown: ledger is closed")

// Ledger 是記憶體裡的寄送紀錄簿，代替真正的資料庫，只保留關機看得到的兩個性質：
// 關閉之後寫不進去；ctx 已經結束的寫入會失敗（同 database/sql 的 ExecContext）。
type Ledger struct {
	mu     sync.Mutex
	closed bool
	states map[string]State
}

// NewLedger 回傳空的紀錄簿。
func NewLedger() *Ledger {
	return &Ledger{states: map[string]State{}}
}

// Begin 在送出之前寫下「寄送中」的紀錄。
func (l *Ledger) Begin(ctx context.Context, key string) error {
	return l.write(ctx, key, StateSending)
}

// Finish 把送出的結果記進紀錄。
func (l *Ledger) Finish(ctx context.Context, key string, s State) error {
	return l.write(ctx, key, s)
}

func (l *Ledger) write(ctx context.Context, key string, s State) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return ErrClosed
	}
	l.states[key] = s
	return nil
}

// Close 關閉紀錄簿；之後的寫入都回傳 ErrClosed。
func (l *Ledger) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.closed = true
}

// State 回傳一則提醒目前的狀態，沒有紀錄時 ok 為 false。關閉之後仍可讀，讓呼叫端檢查紀錄。
func (l *Ledger) State(key string) (s State, ok bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	s, ok = l.states[key]
	return s, ok
}

// Sender 送出一則通知。
type Sender interface {
	// Send 送出 key 代表的通知；ctx 結束或逾時就放棄。
	Send(ctx context.Context, key string) error
}

// Service 是縮小版的提醒服務：依序處理一批到期的提醒，直到收到結束訊號。
type Service struct {
	// Ledger 是寄送紀錄簿。
	Ledger *Ledger
	// Sender 是推播的送出。
	Sender Sender
	// Concurrency 是同時進行中的送出上限，至少 1。
	Concurrency int
	// RequestTimeout 是每個請求的逾時；before 沒有用到。
	RequestTimeout time.Duration
	// ShutdownTimeout 是收到結束訊號之後整個關機的期限；before 沒有用到。
	ShutdownTimeout time.Duration
}

// ANCHOR: run

// Run 在背景處理 due，ctx 結束（收到 SIGTERM）就關紀錄簿、回傳。
// 刻意保留的問題：沒有人等 scan 結束，也沒有期限；關紀錄簿在進行中的送出之前。
func (s *Service) Run(ctx context.Context, due []string) error {
	go s.scan(ctx, due)
	<-ctx.Done()
	s.Ledger.Close()
	return nil
}

// ANCHOR_END: run

// scan 處理 due，同時進行的送出不超過 Concurrency；ctx 結束就不再開始新的一則。
func (s *Service) scan(ctx context.Context, due []string) {
	var g errgroup.Group
	g.SetLimit(s.Concurrency)
	for _, key := range due {
		if ctx.Err() != nil {
			break
		}
		g.Go(func() error {
			s.deliver(ctx, key)
			return nil
		})
	}
	_ = g.Wait()
}

// ANCHOR: deliver

// deliver 處理一則提醒：寫紀錄、送出、記結果。
// 刻意保留的問題：ctx 直接傳給送出與寫入，結束訊號一來，送出被切斷、結果寫不進去，
// 紀錄停在寄送中；錯誤也被丟掉，沒有人知道。
func (s *Service) deliver(ctx context.Context, key string) {
	if err := s.Ledger.Begin(ctx, key); err != nil {
		return
	}
	state := StateSent
	if err := s.Sender.Send(ctx, key); err != nil {
		state = StateRetrying
	}
	_ = s.Ledger.Finish(ctx, key, state)
}

// ANCHOR_END: deliver
