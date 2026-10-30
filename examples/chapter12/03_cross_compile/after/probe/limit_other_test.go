//go:build !unix

package probe_test

import (
	"errors"
	"testing"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter12/03_cross_compile/after/probe"
)

func TestFileLimitUnsupported(t *testing.T) {
	if _, err := probe.FileLimit(); !errors.Is(err, errors.ErrUnsupported) {
		t.Errorf("got %v, want ErrUnsupported", err)
	}
}

// 不支援不是錯誤：Collect 照樣成功，FileLimit 留 0。
func TestCollectWithoutFileLimit(t *testing.T) {
	info, err := probe.Collect()
	if err != nil {
		t.Fatal(err)
	}
	if info.FileLimit != 0 {
		t.Errorf("FileLimit = %d, want 0", info.FileLimit)
	}
}
