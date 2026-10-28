// Package cpucheck 在服務啟動時檢查 GOMAXPROCS 是否還跟著 cgroup 的 CPU 配額走。
package cpucheck

import (
	"errors"
	"fmt"
	"runtime/debug"
	"strconv"
	"strings"
)

// ANCHOR: check

// Check 只讀不改：回報執行檔與環境裡有哪些設定，會讓 runtime 不再依 cgroup 的
// CPU 配額決定 GOMAXPROCS 並定期更新。沒有這類設定時回傳 nil。
//
// info 通常來自 debug.ReadBuildInfo，getenv 通常是 os.Getenv。
func Check(info *debug.BuildInfo, getenv func(string) string) error {
	var errs []error
	if v := getenv("GOMAXPROCS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			errs = append(errs, fmt.Errorf("GOMAXPROCS=%s is set in the environment: the runtime keeps it instead of following the cgroup CPU limit", v))
		}
	}

	// 兩個來源：編譯時由 go.mod 的 go 版本決定的預設值，以及執行時的 GODEBUG。
	lists := []string{getenv("GODEBUG")}
	for _, s := range info.Settings {
		if s.Key == "DefaultGODEBUG" {
			lists = append(lists, s.Value)
		}
	}
	for _, list := range lists {
		for kv := range strings.SplitSeq(list, ",") {
			if kv == "containermaxprocs=0" || kv == "updatemaxprocs=0" {
				errs = append(errs, fmt.Errorf("GODEBUG %s turns off the cgroup-aware default: use go 1.25 or later in go.mod and drop the setting", kv))
			}
		}
	}
	return errors.Join(errs...)
}

// ANCHOR_END: check
