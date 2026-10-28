package orders_test

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter5/03_fake_stub_mock/before/orders"
)

// ANCHOR: mock

// mock 一個物件同時扮演 Catalog、Store 與 Mailer，
// 把每次呼叫（含參數）依序記進 calls，測試最後逐項比對。
type mock struct {
	calls []string
}

func (m *mock) record(format string, args ...any) {
	m.calls = append(m.calls, fmt.Sprintf(format, args...))
}

func (m *mock) Price(_ context.Context, sku string) (int, error) {
	m.record("Price(%s)", sku)
	return map[string]int{"apple": 120, "pear": 80}[sku], nil
}

func (m *mock) Save(_ context.Context, o orders.Order) (string, error) {
	m.record("Save(%s, total=%d)", o.Customer, o.Total)
	return "o-1", nil
}

func (m *mock) Send(_ context.Context, to, subject string) error {
	m.record("Send(%s, %q)", to, subject)
	return nil
}

// verify 要求實際的呼叫與 want 的內容、順序完全一致。
func (m *mock) verify(want ...string) error {
	if slices.Equal(m.calls, want) {
		return nil
	}
	return fmt.Errorf("calls differ\ngot:  %s\nwant: %s", strings.Join(m.calls, " "), strings.Join(want, " "))
}

// ANCHOR_END: mock
