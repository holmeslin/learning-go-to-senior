package cpucheck_test

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"
	"testing"

	cpucheck "github.com/holmeslin/learning-go-to-senior/examples/chapter8/05_container_gomaxprocs/after"
)

// ANCHOR: helpers

// buildInfo 造一份 DefaultGODEBUG 為 godebug 的編譯資訊。
func buildInfo(godebug string) *debug.BuildInfo {
	return &debug.BuildInfo{Settings: []debug.BuildSetting{{Key: "DefaultGODEBUG", Value: godebug}}}
}

// env 造一個只有 kv 的假環境。
func env(kv map[string]string) func(string) string {
	return func(key string) string { return kv[key] }
}

// ANCHOR_END: helpers

// ANCHOR: table

func TestCheck(t *testing.T) {
	tests := []struct {
		name string
		info *debug.BuildInfo
		env  map[string]string
		want []string // 錯誤訊息應該含有的片段；空表示沒有錯誤
	}{
		{name: "defaults", info: buildInfo("")},
		{name: "unrelated godebug", info: buildInfo("tlssha1=1,urlstrictcolons=0")},
		{name: "env ignored when not positive", info: buildInfo(""), env: map[string]string{"GOMAXPROCS": "0"}},
		{name: "env set", info: buildInfo(""), env: map[string]string{"GOMAXPROCS": "8"}, want: []string{"GOMAXPROCS=8"}},
		{name: "old go.mod", info: buildInfo("containermaxprocs=0,updatemaxprocs=0"), want: []string{"containermaxprocs=0", "updatemaxprocs=0"}},
		{name: "godebug env", info: buildInfo(""), env: map[string]string{"GODEBUG": "updatemaxprocs=0"}, want: []string{"updatemaxprocs=0"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cpucheck.Check(tt.info, env(tt.env))
			if len(tt.want) == 0 {
				if err != nil {
					t.Fatalf("Check() = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("Check() = nil, want an error containing %q", tt.want)
			}
			for _, w := range tt.want {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("Check() = %q, want it to contain %q", err, w)
				}
			}
		})
	}
}

// ANCHOR_END: table

// ANCHOR: build

// 這個 module 的 go.mod 是 go 1.27，編譯出來的執行檔不該帶著關掉自動調整的預設值。
func TestCheckThisBuild(t *testing.T) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		t.Skip("no build info")
	}
	if err := cpucheck.Check(info, env(nil)); err != nil {
		t.Errorf("Check() = %v, want nil", err)
	}
}

// ANCHOR_END: build

// ANCHOR: readonly

// 啟動時的檢查只讀不改：先把 GOMAXPROCS 設成哨兵值，檢查完必須原封不動。
func TestCheckDoesNotOverride(t *testing.T) {
	prev := runtime.GOMAXPROCS(3)
	t.Cleanup(func() { runtime.GOMAXPROCS(prev) })

	_ = cpucheck.Check(buildInfo("containermaxprocs=0"), env(map[string]string{"GOMAXPROCS": "8"}))
	if got := runtime.GOMAXPROCS(0); got != 3 {
		t.Errorf("GOMAXPROCS(0) = %d after Check, want 3", got)
	}
}

// ANCHOR_END: readonly

// ANCHOR: example

func ExampleCheck() {
	info := buildInfo("containermaxprocs=0")
	fmt.Println(cpucheck.Check(info, env(map[string]string{"GOMAXPROCS": "8"})))
	fmt.Println(cpucheck.Check(buildInfo(""), env(nil)))
	// Output:
	// GOMAXPROCS=8 is set in the environment: the runtime keeps it instead of following the cgroup CPU limit
	// GODEBUG containermaxprocs=0 turns off the cgroup-aware default: use go 1.25 or later in go.mod and drop the setting
	// <nil>
}

// ANCHOR_END: example
