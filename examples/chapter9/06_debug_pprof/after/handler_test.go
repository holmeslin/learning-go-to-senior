package stock_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	stock "github.com/holmeslin/learning-go-to-senior/examples/chapter9/06_debug_pprof/after"
)

// ANCHOR: public

func TestPublicHandlerHasNoDiagnostics(t *testing.T) {
	t.Parallel()
	public := stock.NewPublicHandler()
	for _, path := range []string{
		"/debug/pprof/",
		"/debug/pprof/heap",
		"/debug/pprof/goroutine?debug=1",
		"/debug/pprof/profile?seconds=1",
		"/debug/pprof/symbol",
		"/debug/pprof/trace?seconds=1",
		"/debug/pprof/cmdline",
	} {
		if got := status(public, path); got != http.StatusNotFound {
			t.Errorf("public GET %s = %d, want 404", path, got)
		}
	}
}

// ANCHOR_END: public

func TestAdminHandlerListsProfiles(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	stock.NewAdminHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("admin GET /debug/pprof/ = %d, want 200", rec.Code)
	}
	for _, name := range []string{"goroutine", "heap", "allocs", "profile"} {
		if !strings.Contains(rec.Body.String(), name) {
			t.Errorf("profile list does not mention %q", name)
		}
	}
}

func TestAdminHandlerServesOnlyChosenEndpoints(t *testing.T) {
	t.Parallel()
	admin := stock.NewAdminHandler()
	for path, want := range map[string]int{
		"/debug/pprof/goroutine?debug=1": http.StatusOK,
		"/debug/pprof/heap":              http.StatusOK,
		"/debug/pprof/symbol":            http.StatusOK,
		"/debug/pprof/cmdline":           http.StatusNotFound, // 命令列可能帶著密碼，所以沒有註冊
		"/stock/A-100":                   http.StatusNotFound, // 管理用的 handler 沒有業務路由
	} {
		if got := status(admin, path); got != want {
			t.Errorf("admin GET %s = %d, want %d", path, got, want)
		}
	}
}

// ANCHOR: defaultmux

// 只要 import 了 net/http/pprof，DefaultServeMux 就有診斷端點，
// 所以對外的 handler 不能是 DefaultServeMux。
func TestDefaultServeMuxStillHasPprof(t *testing.T) {
	t.Parallel()
	if got := status(http.DefaultServeMux, "/debug/pprof/"); got != http.StatusOK {
		t.Errorf("DefaultServeMux GET /debug/pprof/ = %d, want 200", got)
	}
}

// ANCHOR_END: defaultmux
