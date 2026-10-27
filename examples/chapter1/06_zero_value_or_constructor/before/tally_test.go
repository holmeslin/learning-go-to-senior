package tally_test

import (
	"io"
	"testing"

	tally "github.com/holmeslin/learning-go-to-senior/examples/chapter1/06_zero_value_or_constructor/before"
)

// 這個測試記錄的是 before 的缺陷：兩個型別的零值一用就 panic。
// ANCHOR: test
func TestZeroValuePanics(t *testing.T) {
	tests := []struct {
		name string
		use  func()
	}{
		{"Counter", func() {
			var c tally.Counter
			c.Add("/home")
		}},
		{"Reporter", func() {
			c := tally.NewCounter()
			c.Add("/home")
			var r tally.Reporter
			r.Report(c)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("零值沒有 panic")
				}
			}()
			tt.use()
		})
	}
}

// ANCHOR_END: test

func TestConstructorsWork(t *testing.T) {
	c := tally.NewCounter()
	c.Add("/home")
	tally.NewReporter(io.Discard).Report(c)
	if c.Count("/home") != 1 {
		t.Fatal("Count != 1")
	}
}
