// Package notify 寄通知信：歡迎信排進佇列由 Run 寄出，也能一次寄給一批收件人。
package notify

import (
	"context"
	"log"
	"sync"
)

// SendFunc 寄一封信給 to，失敗時回傳錯誤。
type SendFunc func(to string) error

// Notifier 寄信。它自己不開長期的 goroutine，要不要背景執行由呼叫 Run 的人決定。
type Notifier struct {
	send  SendFunc
	queue chan string
}

// New 建立 Notifier。
func New(send SendFunc) *Notifier {
	return &Notifier{send: send, queue: make(chan string, 16)}
}

// Welcome 把 to 排進佇列，等 Run 寄出。佇列滿時會等到 Run 取走一封。
func (n *Notifier) Welcome(to string) {
	n.queue <- to
}

// ANCHOR: run

// Run 依序寄出佇列裡的信，直到 ctx 結束才回傳。
// 回傳時已經沒有寄信的工作在進行：正在寄的那一封會寄完，佇列裡剩下的不會再寄。
func (n *Notifier) Run(ctx context.Context) {
	for ctx.Err() == nil {
		select {
		case <-ctx.Done():
		case to := <-n.queue:
			if err := n.send(to); err != nil {
				log.Printf("welcome %s: %v", to, err)
			}
		}
	}
}

// ANCHOR_END: run

// ANCHOR: broadcast

// Broadcast 對每個收件人各開一個 goroutine 寄信，全部寄完才回傳。
// 回傳寄失敗的收件人，順序與 recipients 相同。
func (n *Notifier) Broadcast(recipients []string) []string {
	failed := make([]bool, len(recipients))
	var wg sync.WaitGroup
	for i, to := range recipients {
		wg.Go(func() { failed[i] = n.send(to) != nil })
	}
	wg.Wait()

	var out []string
	for i, to := range recipients {
		if failed[i] {
			out = append(out, to)
		}
	}
	return out
}

// ANCHOR_END: broadcast
