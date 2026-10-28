package route_test

import (
	"strings"
	"testing"

	route "github.com/holmeslin/learning-go-to-senior/examples/chapter7/08_perf_regression_ci/after"
)

// ANCHOR: reference

// reference 是 Template 改寫之前常見的寫法：切開、逐段替換、再接起來。
func reference(path string) string {
	parts := strings.Split(path, "/")
	for i, p := range parts {
		if p != "" && strings.Trim(p, "0123456789") == "" {
			parts[i] = ":id"
		}
	}
	return strings.Join(parts, "/")
}

// ANCHOR_END: reference

var paths = []string{
	"",
	"/",
	"/healthz",
	"/users/7",
	"/users/48213/orders/90177/items/3",
	"/users/48213/profile",
	"/v2/users/abc123/orders/",
	"//1//2",
}

func TestTemplateMatchesReference(t *testing.T) {
	for _, p := range paths {
		if got, want := route.Template(p), reference(p); got != want {
			t.Errorf("Template(%q) = %q, want %q", p, got, want)
		}
	}
}

// ANCHOR: allocs

func TestTemplateAllocs(t *testing.T) {
	for _, p := range paths {
		allocs := testing.AllocsPerRun(100, func() { route.Template(p) })
		if allocs > 1 {
			t.Errorf("Template(%q) allocs = %v, want at most 1", p, allocs)
		}
	}
}

// ANCHOR_END: allocs

// 這個測試確認上面的門檻擋得住 reference 這種寫法。
func TestAllocsGuardCatchesReference(t *testing.T) {
	p := "/users/48213/orders/90177/items/3"
	if allocs := testing.AllocsPerRun(100, func() { reference(p) }); allocs <= 1 {
		t.Errorf("reference(%q) allocs = %v, want more than 1", p, allocs)
	}
}

func BenchmarkTemplate(b *testing.B) {
	p := "/users/48213/orders/90177/items/3"
	for b.Loop() {
		route.Template(p)
	}
}
