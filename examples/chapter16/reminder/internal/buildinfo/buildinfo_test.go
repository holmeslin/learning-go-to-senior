package buildinfo_test

import (
	"runtime/debug"
	"testing"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter16/reminder/internal/buildinfo"
)

// ANCHOR: table

// fake 是假的 debug.BuildInfo：go test 的測試執行檔沒有 VCS 資訊，測試不去讀真的 ReadBuildInfo。
func fake(mainVersion string, settings ...debug.BuildSetting) *debug.BuildInfo {
	return &debug.BuildInfo{GoVersion: "go1.27.0", Main: debug.Module{Version: mainVersion}, Settings: settings}
}

// 注入的版本號優先；沒注入就用建置資訊；什麼都沒有也不會是空字串。
func TestFrom(t *testing.T) {
	t.Parallel()
	vcs := []debug.BuildSetting{
		{Key: "vcs.revision", Value: "7a0c240b07b41e3a3f1b15d168f8056f5af39e0e"},
		{Key: "vcs.time", Value: "2030-06-10T08:00:00Z"},
		{Key: "vcs.modified", Value: "true"},
	}
	tests := []struct {
		name     string
		bi       *debug.BuildInfo
		injected string
		want     buildinfo.Info
	}{
		{
			name:     "injected version wins, vcs facts still come from the build",
			bi:       fake("v1.4.1-0.20300610080000-7a0c240b07b4+dirty", vcs...),
			injected: "v1.5.0",
			want: buildinfo.Info{
				Version: "v1.5.0", Revision: "7a0c240b07b41e3a3f1b15d168f8056f5af39e0e",
				Time: "2030-06-10T08:00:00Z", Modified: true, GoVersion: "go1.27.0",
			},
		},
		{
			name: "not injected: the main module version",
			bi:   fake("v1.4.1-0.20300610080000-7a0c240b07b4+dirty", vcs...),
			want: buildinfo.Info{
				Version: "v1.4.1-0.20300610080000-7a0c240b07b4+dirty", Revision: "7a0c240b07b41e3a3f1b15d168f8056f5af39e0e",
				Time: "2030-06-10T08:00:00Z", Modified: true, GoVersion: "go1.27.0",
			},
		},
		{
			name: "go run: no vcs info",
			bi:   fake("(devel)"),
			want: buildinfo.Info{Version: "(devel)", GoVersion: "go1.27.0"},
		},
		{
			name:     "no build info at all, injected version only",
			injected: "v1.5.0",
			want:     buildinfo.Info{Version: "v1.5.0"},
		},
		{
			name: "no build info and nothing injected",
			want: buildinfo.Info{Version: "unknown"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := buildinfo.From(tt.bi, tt.injected); got != tt.want {
				t.Errorf("From() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// ANCHOR_END: table

func TestInfoString(t *testing.T) {
	t.Parallel()
	const rev = "7a0c240b07b41e3a3f1b15d168f8056f5af39e0e"
	tests := []struct {
		name string
		info buildinfo.Info
		want string
	}{
		{"clean", buildinfo.Info{Version: "v1.5.0", Revision: rev, Time: "2030-06-10T08:00:00Z", GoVersion: "go1.27.0"}, "v1.5.0 7a0c240b07b4 2030-06-10T08:00:00Z go1.27.0"},
		{"dirty", buildinfo.Info{Version: "v1.5.0", Revision: rev, Time: "2030-06-10T08:00:00Z", Modified: true, GoVersion: "go1.27.0"}, "v1.5.0 7a0c240b07b4-dirty 2030-06-10T08:00:00Z go1.27.0"},
		{"no vcs info", buildinfo.Info{Version: "v1.5.0", GoVersion: "go1.27.0"}, "v1.5.0 (no vcs info) go1.27.0"},
	}
	for _, tt := range tests {
		if got := tt.info.String(); got != tt.want {
			t.Errorf("%s: String() = %q, want %q", tt.name, got, tt.want)
		}
	}
}
