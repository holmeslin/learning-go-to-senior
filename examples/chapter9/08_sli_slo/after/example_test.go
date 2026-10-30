package slo_test

import (
	"fmt"
	"time"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"

	slo "github.com/holmeslin/learning-go-to-senior/examples/chapter9/08_sli_slo/after"
)

// ANCHOR: example

func ExampleLatencyView() {
	ms := time.Millisecond
	for _, c := range []struct {
		name string
		tail time.Duration
	}{
		{"A", 260 * ms},
		{"B", 400 * ms},
	} {
		rm, err := collect([]traffic{{200, 120 * ms, 96}, {200, c.tail, 4}}, sdkmetric.WithView(slo.LatencyView()))
		if err != nil {
			panic(err)
		}
		report, err := slo.Compute(rm)
		if err != nil {
			panic(err)
		}
		fmt.Printf("%s: latency SLI %.2f, %.0f of %d requests may be slow\n",
			c.name, report.Latency.SLI, report.Latency.Budget, report.Latency.Total)
	}
	// Output:
	// A: latency SLI 1.00, 1 of 100 requests may be slow
	// B: latency SLI 0.96, 1 of 100 requests may be slow
}

// ANCHOR_END: example

// ANCHOR: burn

func ExampleResult_ShouldPage() {
	ms := time.Millisecond
	for _, w := range []struct {
		name string
		reqs []traffic
	}{
		{"quiet", []traffic{{200, 120 * ms, 9999}, {500, 120 * ms, 1}}},
		{"burst", []traffic{{200, 120 * ms, 9800}, {503, 50 * ms, 200}}},
	} {
		rm, err := collect(w.reqs, sdkmetric.WithView(slo.LatencyView()))
		if err != nil {
			panic(err)
		}
		report, err := slo.Compute(rm)
		if err != nil {
			panic(err)
		}
		a := report.Availability
		fmt.Printf("%s: sli %.4f, budget left %.0f of %.0f, burn %.1fx, page %t\n",
			w.name, a.SLI, a.Remaining, a.Budget, a.BurnRate, a.ShouldPage())
	}
	// Output:
	// quiet: sli 0.9999, budget left 9 of 10, burn 0.1x, page false
	// burst: sli 0.9800, budget left -190 of 10, burn 20.0x, page true
}

// ANCHOR_END: burn
