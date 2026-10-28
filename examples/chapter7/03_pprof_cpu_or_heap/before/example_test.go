package logstats_test

import (
	"fmt"
	"maps"
	"slices"

	logstats "github.com/holmeslin/learning-go-to-senior/examples/chapter7/03_pprof_cpu_or_heap/before"
)

func ExampleSummarize() {
	stats, err := logstats.Summarize([]string{
		"2026-10-28T10:00:00Z GET /api/users/42 200 12",
		"2026-10-28T10:00:01Z GET /api/users/7 200 30",
		"2026-10-28T10:00:02Z GET /api/users/42 503 80",
		"2026-10-28T10:00:03Z GET /healthz 200 1",
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, path := range slices.Sorted(maps.Keys(stats)) {
		s := stats[path]
		fmt.Printf("%s count=%d errors=%d p95=%dms last=%s\n", path, s.Count, s.Errors, s.P95, s.Last.Format("15:04:05"))
	}
	// Output:
	// /api/users/:id count=3 errors=1 p95=80ms last=10:00:02
	// /healthz count=1 errors=0 p95=1ms last=10:00:03
}
