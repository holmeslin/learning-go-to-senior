package runtimestats_test

import (
	"fmt"
	"maps"
	"slices"

	runtimestats "github.com/holmeslin/learning-go-to-senior/examples/chapter8/06_runtime_metrics/after"
)

// ANCHOR: example

func ExampleSampler_Read() {
	s := runtimestats.New(
		"/gc/heap/goal:bytes",
		"/gc/cycles/total:gc-cycles",
		"/sched/latencies:seconds",
		"/no/such/metric:bytes",
	)
	dst := map[string]float64{}
	s.Read(dst)

	for _, name := range slices.Sorted(maps.Keys(dst)) {
		fmt.Println("read:", name)
	}
	for _, name := range s.Unsupported() {
		fmt.Println("unsupported:", name)
	}
	// Output:
	// read: /gc/cycles/total:gc-cycles
	// read: /gc/heap/goal:bytes
	// unsupported: /sched/latencies:seconds
	// unsupported: /no/such/metric:bytes
}

// ANCHOR_END: example
