// Package memlimit 由 cgroup v2 的 memory.max 算出 GOMEMLIMIT，讓 GC 知道容器的記憶體上限。
package memlimit

import (
	"errors"
	"fmt"
	"io/fs"
	"math"
	"runtime/debug"
	"strconv"
	"strings"
)

// ANCHOR: limit

// ErrNoLimit 表示 memory.max 的內容是 max，也就是 cgroup 沒有設記憶體上限。
var ErrNoLimit = errors.New("no memory limit")

// Limit 讀 fsys 根目錄下的 memory.max，回傳該設成 GOMEMLIMIT 的位元組數：
// 上限的 90%，其餘 10% 留給 runtime 的記憶體限制算不到的部分。
// 內容是 max 時回傳 [ErrNoLimit]；檔案不存在或格式不對時回傳錯誤。
func Limit(fsys fs.FS) (int64, error) {
	data, err := fs.ReadFile(fsys, "memory.max")
	if err != nil {
		return 0, fmt.Errorf("read memory limit: %w", err)
	}
	s := strings.TrimSpace(string(data))
	if s == "max" {
		return 0, ErrNoLimit
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse memory.max %q: %w", s, err)
	}
	if n <= 0 {
		return 0, fmt.Errorf("invalid memory.max %d", n)
	}
	return n - n/10, nil
}

// ANCHOR_END: limit

// ANCHOR: apply

// ErrAlreadySet 表示行程已經有 memory limit（例如部署設定了 GOMEMLIMIT），Apply 不會覆蓋它。
var ErrAlreadySet = errors.New("memory limit already set")

// Apply 依 [Limit] 的結果呼叫 [debug.SetMemoryLimit]，回傳設定的值。
// 行程已經有 memory limit 時什麼都不做，回傳 [ErrAlreadySet]，讓部署設定的 GOMEMLIMIT 優先。
func Apply(fsys fs.FS) (int64, error) {
	if debug.SetMemoryLimit(-1) != math.MaxInt64 {
		return 0, ErrAlreadySet
	}
	limit, err := Limit(fsys)
	if err != nil {
		return 0, err
	}
	debug.SetMemoryLimit(limit)
	return limit, nil
}

// ANCHOR_END: apply
