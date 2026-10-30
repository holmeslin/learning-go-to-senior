package version_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter12/01_build_version_info/after/version"
)

// ANCHOR: example

// 同一份假的 BuildInfo：先印成一行（-version 旗標與日誌用），
// 再看版本端點回的 JSON。
func Example() {
	info := version.FromBuildInfo(fakeBuildInfo("v1.4.0", vcsSettings("true")...), "ci-1234")
	fmt.Println(info)

	rec := httptest.NewRecorder()
	version.Handler(info).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/version", nil))
	fmt.Print(rec.Body)
	// Output:
	// v1.4.0 0123456789ab-dirty 2026-10-30T08:00:00Z go1.27.0 build ci-1234
	// {"version":"v1.4.0","revision":"0123456789abcdef0123456789abcdef01234567","time":"2026-10-30T08:00:00Z","modified":true,"go_version":"go1.27.0","build":"ci-1234"}
}

// ANCHOR_END: example
