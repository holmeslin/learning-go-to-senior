package rules_test

import (
	"fmt"
	"testing"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter7/07_pgo/after/rules"
)

func TestTally(t *testing.T) {
	reqs := []rules.Request{
		{Path: "/api/users", Status: 200, Bytes: 10},
		{Path: "/API/orders", Status: 503, Bytes: 20},
		{Path: "/static/app.js", Status: 200, Bytes: 30},
		{Path: "/ap", Status: 500, Bytes: 40},
	}
	tests := []struct {
		name string
		rule rules.Rule
		want rules.Stats
	}{
		{"路徑前綴忽略大小寫", &rules.PathPrefix{Prefix: "/api/"}, rules.Stats{Total: 4, Matched: 2, Bytes: 30}},
		{"5xx", &rules.ServerError{}, rules.Stats{Total: 4, Matched: 2, Bytes: 60}},
		{"空前綴全部符合", &rules.PathPrefix{}, rules.Stats{Total: 4, Matched: 4, Bytes: 100}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := rules.Tally(reqs, tt.rule); got != tt.want {
				t.Errorf("Tally() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestParseRule(t *testing.T) {
	rule, err := rules.ParseRule("path:/api/")
	if err != nil {
		t.Fatal(err)
	}
	if got := rules.Tally([]rules.Request{{Path: "/API/x"}, {Path: "/x"}}, rule).Matched; got != 1 {
		t.Errorf("path rule matched %d, want 1", got)
	}
	if _, err := rules.ParseRule("5xx"); err != nil {
		t.Errorf("ParseRule(5xx) error = %v", err)
	}
	if _, err := rules.ParseRule("bogus"); err == nil {
		t.Error("ParseRule(bogus) error = nil, want an error")
	}
}

// ANCHOR: allocs

// 熱路徑每次呼叫都不該配置；PGO 只改機器碼，不會改變這件事，所以這個斷言有沒有 profile 都成立。
func TestTallyDoesNotAllocate(t *testing.T) {
	reqs := rules.Sample(1000)
	for _, rule := range []rules.Rule{&rules.PathPrefix{Prefix: "/api/"}, &rules.ServerError{}} {
		allocs := testing.AllocsPerRun(10, func() { rules.Tally(reqs, rule) })
		if allocs != 0 {
			t.Errorf("Tally(%T) allocs = %v, want 0", rule, allocs)
		}
	}
}

// ANCHOR_END: allocs

// ANCHOR: example

func ExampleTally() {
	reqs := rules.Sample(1000)
	fmt.Printf("%+v\n", rules.Tally(reqs, &rules.PathPrefix{Prefix: "/api/"}))
	fmt.Printf("%+v\n", rules.Tally(reqs, &rules.ServerError{}))
	// Output:
	// {Total:1000 Matched:700 Bytes:87200}
	// {Total:1000 Matched:50 Bytes:6350}
}

// ANCHOR_END: example

// ANCHOR: bench

func BenchmarkTally(b *testing.B) {
	reqs := rules.Sample(100_000)
	for _, tc := range []struct {
		name string
		rule rules.Rule
	}{
		{"path_prefix", &rules.PathPrefix{Prefix: "/api/"}},
		{"server_error", &rules.ServerError{}},
	} {
		b.Run("rule="+tc.name, func(b *testing.B) {
			for b.Loop() {
				rules.Tally(reqs, tc.rule)
			}
		})
	}
}

// ANCHOR_END: bench
