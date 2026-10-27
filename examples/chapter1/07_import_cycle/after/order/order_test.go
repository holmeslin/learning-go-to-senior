package order_test

import (
	"errors"
	"testing"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter1/07_import_cycle/after/order"
)

// ANCHOR: fake
// fakeNotifier 只靠 order 自己的型別就寫得出來；這個測試不需要 notify。
type fakeNotifier struct {
	got []order.Order
	err error
}

func (f *fakeNotifier) Notify(o order.Order) error {
	f.got = append(f.got, o)
	return f.err
}

// ANCHOR_END: fake

func TestPlaceNotifies(t *testing.T) {
	f := &fakeNotifier{}

	o, err := order.Place(f, 7, "滑鼠", 500)
	if err != nil {
		t.Fatalf("Place: %v", err)
	}
	if len(f.got) != 1 || f.got[0] != o {
		t.Errorf("通知到的訂單 = %v，想要剛成立的 %v", f.got, o)
	}
}

func TestPlaceNotifyFails(t *testing.T) {
	boom := errors.New("smtp down")
	f := &fakeNotifier{err: boom}

	o, err := order.Place(f, 8, "螢幕", 3000)
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v，想要包住 %v", err, boom)
	}
	if o.ID != 8 {
		t.Errorf("通知失敗時仍應回傳已成立的訂單，得到 %v", o)
	}
}
