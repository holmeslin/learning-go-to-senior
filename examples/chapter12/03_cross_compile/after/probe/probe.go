// Package probe 收集程式所在主機的基本資訊，純 Go 實作，可以交叉編譯到任何平台。
package probe

import (
	"errors"
	"fmt"
	"os"
	"runtime"
)

// Info 是一次收集的結果。
type Info struct {
	// OS 是編譯時指定的 GOOS。
	OS string
	// Arch 是編譯時指定的 GOARCH。
	Arch string
	// Hostname 是主機名稱。
	Hostname string
	// FileLimit 是行程能同時開啟的檔案數上限；平台沒有這個概念時為 0。
	FileLimit uint64
}

// String 以「系統/架構 host=主機 nofile=上限」的格式輸出；沒有上限的平台印 n/a。
func (i Info) String() string {
	limit := "n/a"
	if i.FileLimit > 0 {
		limit = fmt.Sprint(i.FileLimit)
	}
	return fmt.Sprintf("%s/%s host=%s nofile=%s", i.OS, i.Arch, i.Hostname, limit)
}

// ANCHOR: collect

// Collect 收集目前主機的資訊。主機名稱用 os.Hostname，不經過 cgo；
// 檔案數上限由各平台自己的 FileLimit 提供，平台不支援時不算錯誤。
func Collect() (Info, error) {
	host, err := os.Hostname()
	if err != nil {
		return Info{}, fmt.Errorf("hostname: %w", err)
	}
	limit, err := FileLimit()
	if err != nil && !errors.Is(err, errors.ErrUnsupported) {
		return Info{}, fmt.Errorf("file limit: %w", err)
	}
	return Info{OS: runtime.GOOS, Arch: runtime.GOARCH, Hostname: host, FileLimit: limit}, nil
}

// ANCHOR_END: collect
