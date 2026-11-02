// Package shutdown 是提醒服務關機順序的縮小版，這個目錄是較好的做法：
// 掃描迴圈歸 Run 管，收到結束訊號後依序停止新的提醒、等進行中的送出、關紀錄簿，
// 全部在同一個關機期限之內；每個請求另有自己的逾時。
//
// Ledger 代替寄送紀錄表，Service.Run 代替服務本體的 Service.Run；不 import 服務本體。
// 常見寫法與它造成的問題在 ../before。
package shutdown

import (
	"context"
	"errors"
	"fmt"
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
	// RequestTimeout 是每個請求的逾時，必須大於零。
	RequestTimeout time.Duration
	// ShutdownTimeout 是收到結束訊號之後整個關機的期限。
	// 它要大於 RequestTimeout 加上餘裕；服務本體的 config.Load 會檢查，這個縮小版不檢查。
	ShutdownTimeout time.Duration
}

// ANCHOR: run

// Run 在背景處理 due，直到 ctx 結束（收到 SIGTERM），再依序關機，期限從收到訊號起算，全部在 ShutdownTimeout 之內：
//
//  1. 停止新的提醒：scan 不再開始新的一則。
//  2. 等進行中的送出做完：每一則最久是一個請求逾時；期限到了仍未結束，回傳錯誤。
//  3. 關紀錄簿：一定在送出都結束之後，進行中的送出才寫得進結果。
//
// 正常關機回傳 nil。
func (s *Service) Run(ctx context.Context, due []string) error {
	scanDone := make(chan struct{})
	var scanErr error
	go func() {
		defer close(scanDone)
		scanErr = s.scan(ctx, due)
	}()

	<-ctx.Done() // 收到結束訊號；scan 已經不再開始新的一則
	// ctx 已經結束，期限要從沒有被取消的 context 另外算。
	grace, cancel := context.WithTimeout(context.WithoutCancel(ctx), s.ShutdownTimeout)
	defer cancel()

	var err error
	select {
	case <-scanDone:
		err = scanErr
	case <-grace.Done():
		err = fmt.Errorf("in-flight sends did not finish: %w", grace.Err())
	}
	s.Ledger.Close()
	return err
}

// ANCHOR_END: run

// scan 處理 due，同時進行的送出不超過 Concurrency；ctx 結束就不再開始新的一則，進行中的做完才回傳。
func (s *Service) scan(ctx context.Context, due []string) error {
	var g errgroup.Group
	g.SetLimit(s.Concurrency)
	for _, key := range due {
		if ctx.Err() != nil {
			break
		}
		g.Go(func() error { return s.deliver(ctx, key) })
	}
	return g.Wait()
}

// ANCHOR: deliver

// deliver 處理一則提醒：寫紀錄、送出、記結果。
func (s *Service) deliver(ctx context.Context, key string) error {
	// 取消只決定要不要開始這一則，對應服務本體 deliver 裡的 limiter.Wait(ctx)：
	// 還沒寫紀錄就放棄，不留下停在寄送中的紀錄。
	if ctx.Err() != nil {
		return nil
	}
	// 從這裡起這一則要做完：切開取消，送出與寫入不被結束訊號切斷。
	ctx = context.WithoutCancel(ctx)
	if err := s.Ledger.Begin(ctx, key); err != nil {
		return fmt.Errorf("begin %s: %w", key, err)
	}

	// 逾時只屬於這個請求；記結果用沒有逾時的 ctx，請求逾時之後還寫得進去。
	rctx, cancel := context.WithTimeout(ctx, s.RequestTimeout)
	defer cancel()
	state := StateSent
	if err := s.Sender.Send(rctx, key); err != nil {
		state = StateRetrying // 縮小版不分類錯誤，失敗一律待重試
	}
	if err := s.Ledger.Finish(ctx, key, state); err != nil {
		return fmt.Errorf("finish %s: %w", key, err)
	}
	return nil
}

// ANCHOR_END: deliver
