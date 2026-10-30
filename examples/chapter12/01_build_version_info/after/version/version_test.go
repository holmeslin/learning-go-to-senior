package version_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"runtime/debug"
	"testing"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter12/01_build_version_info/after/version"
)

// ANCHOR: fake

// fakeBuildInfo 組一份假的 BuildInfo。測試執行檔沒有 VCS 資訊，
// 真的 debug.ReadBuildInfo 在測試裡讀到的內容不確定，所以不拿它來斷言。
func fakeBuildInfo(mainVersion string, vcs ...debug.BuildSetting) *debug.BuildInfo {
	return &debug.BuildInfo{
		GoVersion: "go1.27.0",
		Main:      debug.Module{Path: "example.com/versiond", Version: mainVersion},
		Settings:  vcs,
	}
}

// vcsSettings 是 go build 在 git 工作目錄裡記下的 VCS 設定；
// modified 是 "true" 或 "false"。
func vcsSettings(modified string) []debug.BuildSetting {
	return []debug.BuildSetting{
		{Key: "vcs", Value: "git"},
		{Key: "vcs.revision", Value: "0123456789abcdef0123456789abcdef01234567"},
		{Key: "vcs.time", Value: "2026-10-30T08:00:00Z"},
		{Key: "vcs.modified", Value: modified},
	}
}

// ANCHOR_END: fake

// ANCHOR: format

func TestString(t *testing.T) {
	tests := []struct {
		name  string
		bi    *debug.BuildInfo
		build string
		want  string
	}{
		{"clean", fakeBuildInfo("v1.4.0", vcsSettings("false")...), "",
			"v1.4.0 0123456789ab 2026-10-30T08:00:00Z go1.27.0"},
		{"dirty", fakeBuildInfo("v1.4.0", vcsSettings("true")...), "",
			"v1.4.0 0123456789ab-dirty 2026-10-30T08:00:00Z go1.27.0"},
		{"build from the build system", fakeBuildInfo("v1.4.0", vcsSettings("false")...), "ci-1234",
			"v1.4.0 0123456789ab 2026-10-30T08:00:00Z go1.27.0 build ci-1234"},
		{"no vcs info (go run, test binary)", fakeBuildInfo("(devel)"), "",
			"(devel) (no vcs info) go1.27.0"},
		{"no build info at all", nil, "",
			"unknown (no vcs info)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := version.FromBuildInfo(tt.bi, tt.build).String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

// ANCHOR_END: format

func TestFromBuildInfoFields(t *testing.T) {
	got := version.FromBuildInfo(fakeBuildInfo("v1.4.0", vcsSettings("true")...), "ci-1234")
	want := version.Info{
		Version:   "v1.4.0",
		Revision:  "0123456789abcdef0123456789abcdef01234567",
		Time:      "2026-10-30T08:00:00Z",
		Modified:  true,
		GoVersion: "go1.27.0",
		Build:     "ci-1234",
	}
	if got != want {
		t.Errorf("FromBuildInfo() = %+v, want %+v", got, want)
	}
}

func TestHandler(t *testing.T) {
	want := version.FromBuildInfo(fakeBuildInfo("v1.4.0", vcsSettings("true")...), "ci-1234")
	rec := httptest.NewRecorder()
	version.Handler(want).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/version", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	var got version.Info
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got != want {
		t.Errorf("body = %+v, want %+v", got, want)
	}
}
