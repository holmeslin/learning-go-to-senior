package accesslog_test

import (
	"fmt"
	"testing"

	accesslog "github.com/holmeslin/learning-go-to-senior/examples/chapter7/05_common_hotspots/after"
)

func BenchmarkReport(b *testing.B) {
	for _, n := range []int{100, 1000} {
		b.Run(fmt.Sprintf("entries=%d", n), func(b *testing.B) {
			entries := sample(n)
			for b.Loop() {
				_ = accesslog.Report(entries)
			}
		})
	}
}

// ANCHOR: observe

func BenchmarkObserve(b *testing.B) {
	var m accesslog.Metrics
	paths := make([]string, 64)
	for i := range paths {
		paths[i] = fmt.Sprintf("/api/items/%d", i)
	}
	b.RunParallel(func(pb *testing.PB) {
		for i := 0; pb.Next(); i++ {
			m.Observe("GET", paths[i%len(paths)])
		}
	})
}

// ANCHOR_END: observe
