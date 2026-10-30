package probe_test

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter12/03_cross_compile/after/probe"
)

// ANCHOR: example

// 格式化與平台無關：同一份程式在每個平台印出同樣的格式，不支援的欄位印 n/a。
func ExampleInfo_String() {
	fmt.Println(probe.Info{OS: "linux", Arch: "arm64", Hostname: "web-1", FileLimit: 1024})
	fmt.Println(probe.Info{OS: "windows", Arch: "amd64", Hostname: "laptop"})
	// Output:
	// linux/arm64 host=web-1 nofile=1024
	// windows/amd64 host=laptop nofile=n/a
}

// ANCHOR_END: example

func TestCollectReportsBuildTarget(t *testing.T) {
	info, err := probe.Collect()
	if err != nil {
		t.Fatal(err)
	}
	if info.OS != runtime.GOOS || info.Arch != runtime.GOARCH {
		t.Errorf("got %s/%s, want %s/%s", info.OS, info.Arch, runtime.GOOS, runtime.GOARCH)
	}
}
