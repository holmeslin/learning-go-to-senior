package notify_test

import (
	"context"
	"fmt"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter1/04_exported_api_stability/after/notify"
)

type printSender struct{}

func (printSender) Send(m notify.Message) error {
	fmt.Printf("to=%s priority=%d body=%s\n", m.To, m.Priority, m.Body)
	return nil
}

// ANCHOR: example

func Example() {
	var s notify.Sender = printSender{}

	// v1 的呼叫方式，原封不動。
	_ = notify.Send(s, notify.NewMessage("amy@example.com", "你好"))

	// v1.1 新增的部分：Priority 欄位與 SendContext。
	m := notify.Message{To: "bob@example.com", Body: "緊急", Priority: 1}
	_ = notify.SendContext(context.Background(), s, m)
	// Output:
	// to=amy@example.com priority=0 body=你好
	// to=bob@example.com priority=1 body=緊急
}

// ANCHOR_END: example
