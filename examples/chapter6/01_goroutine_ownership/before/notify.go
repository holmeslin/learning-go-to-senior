// Package notify 寄通知信：歡迎信排進佇列由背景寄出，也能一次寄給一批收件人。
package notify

import "log"

// SendFunc 寄一封信給 to，失敗時回傳錯誤。
type SendFunc func(to string) error

// Notifier 在背景寄信。
type Notifier struct {
	send  SendFunc
	queue chan string
}

// ANCHOR: notifier

// New 建立 Notifier，並在背景啟動寄信迴圈。
func New(send SendFunc) *Notifier {
	n := &Notifier{send: send, queue: make(chan string, 16)}
	go n.loop()
	return n
}

func (n *Notifier) loop() {
	for to := range n.queue {
		if err := n.send(to); err != nil {
			log.Printf("welcome %s: %v", to, err)
		}
	}
}

// Welcome 把 to 排進佇列，由背景的迴圈寄出歡迎信。
func (n *Notifier) Welcome(to string) {
	n.queue <- to
}

// Close 關閉佇列。
func (n *Notifier) Close() {
	close(n.queue)
}

// ANCHOR_END: notifier

// ANCHOR: broadcast

// Broadcast 對每個收件人各開一個 goroutine 寄信，立刻回傳。
func (n *Notifier) Broadcast(recipients []string) {
	for _, to := range recipients {
		go func() {
			if err := n.send(to); err != nil {
				log.Printf("broadcast %s: %v", to, err)
			}
		}()
	}
}

// ANCHOR_END: broadcast
