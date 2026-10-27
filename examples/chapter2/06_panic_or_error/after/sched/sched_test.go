package sched_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter2/06_panic_or_error/after/sched"
)

// ANCHOR: must

// nightly 是寫死在程式裡的常數；寫錯是程式的 bug，要在啟動時就讓程式停下來。
var nightly = sched.MustParse("every 24h")

// ANCHOR_END: must

// ANCHOR: example

func ExampleParse() {
	// 使用者輸入或設定檔來的字串，寫錯是預期中的狀況，所以回傳 error。
	for _, spec := range []string{"every 90s", "evry 5m", "every soon", "every -1m"} {
		s, err := sched.Parse(spec)
		if err != nil {
			fmt.Println(err)
			continue
		}
		fmt.Println(s.Interval)
	}
	// Output:
	// 1m30s
	// parse schedule "evry 5m": want "every <duration>"
	// parse schedule "every soon": time: invalid duration "soon"
	// parse schedule "every -1m": interval must be positive
}

// ANCHOR_END: example

func ExampleMustParse() {
	fmt.Println(nightly.Interval)

	defer func() { fmt.Println("panic:", recover()) }()
	sched.MustParse("evry 5m")
	// Output:
	// 24h0m0s
	// panic: parse schedule "evry 5m": want "every <duration>"
}

func TestParse(t *testing.T) {
	tests := []struct {
		spec    string
		want    time.Duration
		wantErr bool
	}{
		{"every 5m", 5 * time.Minute, false},
		{"every 1h30m", 90 * time.Minute, false},
		{"", 0, true},
		{"every", 0, true},
		{"every 0s", 0, true},
		{"every -1m", 0, true},
		{"every 5", 0, true},
	}
	for _, tt := range tests {
		got, err := sched.Parse(tt.spec)
		if (err != nil) != tt.wantErr {
			t.Errorf("Parse(%q) error = %v, wantErr %v", tt.spec, err, tt.wantErr)
			continue
		}
		if got.Interval != tt.want {
			t.Errorf("Parse(%q).Interval = %v, want %v", tt.spec, got.Interval, tt.want)
		}
	}
}

func TestMustParse(t *testing.T) {
	if got := sched.MustParse("every 5m"); got.Interval != 5*time.Minute {
		t.Errorf("MustParse(valid).Interval = %v, want 5m", got.Interval)
	}

	// 壞掉的字串 panic，而且 panic 的值就是 Parse 回傳的那個 error。
	const bad = "every soon"
	_, want := sched.Parse(bad)
	defer func() {
		got, ok := recover().(error)
		if !ok || got.Error() != want.Error() {
			t.Errorf("MustParse(%q) panicked with %v, want error %q", bad, got, want)
		}
	}()
	sched.MustParse(bad)
	t.Error("MustParse of an invalid spec did not panic")
}
