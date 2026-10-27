package notify_test

import (
	"context"
	"errors"
	"testing"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter1/04_exported_api_stability/after/notify"
)

// ANCHOR: lock

// v1 的呼叫端寫的實作：只有 Send。
type v1Sender struct{}

func (v1Sender) Send(notify.Message) error { return nil }

// 以下任何一行編譯失敗，就代表 v1 的呼叫端也會跟著壞。
var (
	_ func(to, body string) notify.Message      = notify.NewMessage
	_ func(notify.Sender, notify.Message) error = notify.Send
	_ notify.Sender                             = v1Sender{}
)

// ANCHOR_END: lock

type recorder struct {
	sent []notify.Message
}

func (r *recorder) Send(m notify.Message) error {
	r.sent = append(r.sent, m)
	return nil
}

// batchRecorder 另外實作可選的 BatchSender。
type batchRecorder struct {
	recorder
	batches int
}

func (b *batchRecorder) SendBatch(ms []notify.Message) error {
	b.batches++
	b.sent = append(b.sent, ms...)
	return nil
}

func TestSendAllFallsBackToSend(t *testing.T) {
	r := &recorder{}
	ms := []notify.Message{notify.NewMessage("a", "1"), notify.NewMessage("b", "2")}
	if err := notify.SendAll(r, ms); err != nil {
		t.Fatal(err)
	}
	if len(r.sent) != 2 {
		t.Errorf("sent %d messages, want 2", len(r.sent))
	}
}

func TestSendAllPrefersBatchSender(t *testing.T) {
	b := &batchRecorder{}
	ms := []notify.Message{notify.NewMessage("a", "1"), notify.NewMessage("b", "2")}
	if err := notify.SendAll(b, ms); err != nil {
		t.Fatal(err)
	}
	if b.batches != 1 || len(b.sent) != 2 {
		t.Errorf("batches=%d sent=%d, want 1 and 2", b.batches, len(b.sent))
	}
}

func TestSendContextCanceled(t *testing.T) {
	r := &recorder{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := notify.SendContext(ctx, r, notify.NewMessage("a", "1"))
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
	if len(r.sent) != 0 {
		t.Errorf("sent %d messages after cancel, want 0", len(r.sent))
	}
}
