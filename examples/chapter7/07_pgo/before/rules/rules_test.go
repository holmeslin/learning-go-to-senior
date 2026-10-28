package rules_test

import (
	"fmt"
	"testing"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter7/07_pgo/before/rules"
)

func ExampleTally() {
	reqs := rules.Sample(1000)
	fmt.Printf("%+v\n", rules.Tally(reqs, &rules.PathPrefix{Prefix: "/api/"}))
	fmt.Printf("%+v\n", rules.Tally(reqs, &rules.ServerError{}))
	// Output:
	// {Total:1000 Matched:700 Bytes:87200}
	// {Total:1000 Matched:50 Bytes:6350}
}

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
