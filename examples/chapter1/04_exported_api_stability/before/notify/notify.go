// Package notify 把訊息交給 Sender 送出。這是 v1.1 直接改成「理想的樣子」的版本。
package notify

import "context"

// ANCHOR: api

// Message 是一則要送出的訊息。
type Message struct {
	To, Body string
	Priority int
}

// New 建立一則訊息（v1 叫 NewMessage）。
func New(to, body string) Message {
	return Message{To: to, Body: body}
}

// Sender 送出訊息，單則或批次。
type Sender interface {
	Send(m Message) error
	SendBatch(ms []Message) error
}

// Send 用 s 送出 m；ctx 取消時不送。
func Send(ctx context.Context, s Sender, m Message) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return s.Send(m)
}

// ANCHOR_END: api
