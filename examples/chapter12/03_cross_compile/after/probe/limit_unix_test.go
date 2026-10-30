//go:build unix

package probe_test

import (
	"testing"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter12/03_cross_compile/after/probe"
)

// ANCHOR: test

// 只在 unix 系統編譯與執行；其他平台的測試在 limit_other_test.go。
func TestFileLimit(t *testing.T) {
	limit, err := probe.FileLimit()
	if err != nil {
		t.Fatal(err)
	}
	if limit == 0 {
		t.Error("file limit is zero")
	}
}

// ANCHOR_END: test

func TestCollectFileLimit(t *testing.T) {
	info, err := probe.Collect()
	if err != nil {
		t.Fatal(err)
	}
	if info.FileLimit == 0 {
		t.Error("Collect left FileLimit zero on a platform that supports it")
	}
}
