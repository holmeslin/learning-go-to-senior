package orders_test

import (
	"context"
	"fmt"
	"strings"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter5/03_fake_stub_mock/after/orders"
)

// ANCHOR: fake

// fakeStore 用 map 假裝訂單資料表：存進去的訂單，之後讀得回來。
type fakeStore struct {
	saved map[string]orders.Order
}

func newFakeStore() *fakeStore {
	return &fakeStore{saved: map[string]orders.Order{}}
}

func (f *fakeStore) Save(_ context.Context, o orders.Order) (string, error) {
	id := fmt.Sprintf("o-%d", len(f.saved)+1)
	o.ID = id
	f.saved[id] = o
	return id, nil
}

// ANCHOR_END: fake

// ANCHOR: stub

// stubCatalog 對每個商品回固定的價格；沒列出的商品回 ErrUnknownSKU。
type stubCatalog map[string]int

func (s stubCatalog) Price(_ context.Context, sku string) (int, error) {
	if price, ok := s[sku]; ok {
		return price, nil
	}
	return 0, orders.ErrUnknownSKU
}

// failingStore 讓 Save 永遠失敗。
type failingStore struct{ err error }

func (f failingStore) Save(context.Context, orders.Order) (string, error) {
	return "", f.err
}

// ANCHOR_END: stub

// ANCHOR: mock

// recordingMailer 記下每一封寄出的信。寄信是 Place 對外的行為，
// 所以只有這個替身讓測試斷言互動。
type recordingMailer struct {
	sent []mail
}

type mail struct{ to, subject string }

func (m *recordingMailer) Send(_ context.Context, to, subject string) error {
	m.sent = append(m.sent, mail{to: to, subject: subject})
	return nil
}

// ANCHOR_END: mock

// placer 是 Service 的一種算價實作，讓同一組行為測試對每一種都跑。
type placer struct {
	name  string
	place func(*orders.Service, context.Context, string, []orders.Line) (orders.Order, error)
}

func placers() []placer {
	return []placer{
		{"Place", (*orders.Service).Place},
		{"PlaceCached", (*orders.Service).PlaceCached},
	}
}

// catalog 與 lines 是各測試共用的輸入：apple 買了兩次。
func catalog() stubCatalog { return stubCatalog{"apple": 120, "pear": 80} }

func lines() []orders.Line {
	return []orders.Line{{SKU: "apple", Qty: 2}, {SKU: "pear", Qty: 1}, {SKU: "apple", Qty: 1}}
}

func subjects(sent []mail) string {
	var s []string
	for _, m := range sent {
		s = append(s, m.subject)
	}
	return strings.Join(s, "; ")
}
