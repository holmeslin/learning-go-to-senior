package roster_test

import (
	"testing"

	roster "github.com/holmeslin/learning-go-to-senior/examples/chapter8/02_when_to_tune_gogc/after"
)

func BenchmarkHandle(b *testing.B) {
	const members = 100_000
	r := roster.Load(members)
	ids := make([]int, 2000)
	for i := range ids {
		ids[i] = i * 37 % members
	}
	for b.Loop() {
		r.Handle(ids)
	}
}
