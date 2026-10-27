// Package notify 把訊息交給 Sender 送出。v1.1 只新增、不修改 v1 的公開 API。
package notify

import "context"

// Message 是一則要送出的訊息。請用欄位名稱建立（Message{To: ...}），
// 不要依欄位順序寫，未來新增欄位時依順序寫的程式會編譯失敗。
type Message struct {
	To   string
	Body string
	// Priority 是 v1.1 新增的欄位，零值代表一般優先順序。
	Priority int
}

// NewMessage 建立一則訊息。
func NewMessage(to, body string) Message {
	return Message{To: to, Body: body}
}

// Sender 送出單一訊息。介面保持只有 Send；新能力用另外的介面表達。
type Sender interface {
	Send(m Message) error
}

// BatchSender 是 v1.1 新增的可選介面：Sender 若也能一次送多則，就實作它。
type BatchSender interface {
	Sender
	SendBatch(ms []Message) error
}

// Send 用 s 送出 m。
func Send(s Sender, m Message) error {
	return SendContext(context.Background(), s, m)
}

// ANCHOR: sendcontext
// SendContext 是 v1.1 新增的函式：與 Send 相同，但 ctx 取消時不送。
func SendContext(ctx context.Context, s Sender, m Message) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return s.Send(m)
}

// ANCHOR_END: sendcontext

// ANCHOR: sendall
// SendAll 送出多則訊息；s 也實作 BatchSender 時改用 SendBatch。
func SendAll(s Sender, ms []Message) error {
	if b, ok := s.(BatchSender); ok {
		return b.SendBatch(ms)
	}
	for _, m := range ms {
		if err := s.Send(m); err != nil {
			return err
		}
	}
	return nil
}

// ANCHOR_END: sendall
