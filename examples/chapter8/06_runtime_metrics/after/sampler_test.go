package runtimestats_test

import (
	"runtime/metrics"
	"slices"
	"strings"
	"testing"

	runtimestats "github.com/holmeslin/learning-go-to-senior/examples/chapter8/06_runtime_metrics/after"
)

// ANCHOR: kinds

func TestNamesExistWithScalarKind(t *testing.T) {
	kinds := map[string]metrics.ValueKind{}
	for _, d := range metrics.All() {
		kinds[d.Name] = d.Kind
	}
	for _, name := range runtimestats.Names() {
		want := metrics.KindUint64
		if strings.HasSuffix(name, ":cpu-seconds") {
			want = metrics.KindFloat64
		}
		if got, ok := kinds[name]; !ok || got != want {
			t.Errorf("%s: kind = %v (listed: %v), want %v", name, got, ok, want)
		}
	}
}

// ANCHOR_END: kinds

func TestNamesReturnsFreshSlice(t *testing.T) {
	first := runtimestats.Names()
	first[0] = "changed by caller"
	if got := runtimestats.Names()[0]; got == "changed by caller" {
		t.Errorf("Names() shares its slice between calls: %q", got)
	}
}

// ANCHOR: unsupported

func TestUnsupportedNamesAreSkipped(t *testing.T) {
	s := runtimestats.New(
		"/gc/heap/goal:bytes",
		"/sched/latencies:seconds", // 直方圖，不是單一數值
		"/no/such/metric:bytes",    // 這個 Go 版本沒有
	)
	want := []string{"/sched/latencies:seconds", "/no/such/metric:bytes"}
	if got := s.Unsupported(); !slices.Equal(got, want) {
		t.Fatalf("Unsupported() = %v, want %v", got, want)
	}

	dst := map[string]float64{}
	s.Read(dst)
	if _, ok := dst["/gc/heap/goal:bytes"]; !ok || len(dst) != 1 {
		t.Errorf("Read filled %v, want only /gc/heap/goal:bytes", dst)
	}
}

// ANCHOR_END: unsupported

func TestReadFillsEveryName(t *testing.T) {
	s := runtimestats.New(runtimestats.Names()...)
	if got := s.Unsupported(); len(got) != 0 {
		t.Fatalf("Unsupported() = %v, want none", got)
	}
	dst := map[string]float64{}
	s.Read(dst)
	for _, name := range runtimestats.Names() {
		if v, ok := dst[name]; !ok || v < 0 {
			t.Errorf("%s = %v (present: %v), want a non-negative value", name, v, ok)
		}
	}
}

// ANCHOR: allocs

func TestReadReusesSamples(t *testing.T) {
	s := runtimestats.New(runtimestats.Names()...)
	dst := map[string]float64{}
	s.Read(dst) // 第一次讀會把 dst 填滿

	if got := testing.AllocsPerRun(100, func() { s.Read(dst) }); got != 0 {
		t.Errorf("Read allocated %v times per run, want 0", got)
	}
}

// ANCHOR_END: allocs
