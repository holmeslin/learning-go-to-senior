// Package buildinfo 整理執行檔的版本資訊：發布流程用 -ldflags 注入的版本號，
// 沒有注入時用 Go 記在執行檔裡的建置資訊（debug.ReadBuildInfo）。
//
// 診斷端點的 /version 與啟動日誌都用同一份 Info，值班的人看到的就是「現在跑的是哪一個建置」。
package buildinfo

import (
	"runtime/debug"
	"strings"
)

// ANCHOR: info

// Info 是一個執行檔的版本資訊，由 [From] 整理而來。
type Info struct {
	// Version 是版本號：-ldflags 注入的值優先，沒注入時是主模組的版本；
	// go run 與測試執行檔是 "(devel)"，讀不到任何建置資訊時是 "unknown"。
	Version string `json:"version"`
	// Revision 是建置時的 commit 雜湊（vcs.revision），沒有 VCS 資訊時為空。
	Revision string `json:"revision,omitempty"`
	// Time 是該 commit 的時間（vcs.time，RFC 3339），沒有 VCS 資訊時為空。
	Time string `json:"time,omitempty"`
	// Modified 表示建置時工作目錄有未提交的修改（vcs.modified）。
	Modified bool `json:"modified"`
	// GoVersion 是建置用的 Go 版本。
	GoVersion string `json:"go_version,omitempty"`
}

// From 把 bi 整理成 [Info]；injected 是 -ldflags -X 注入的版本號，沒注入就傳空字串。
// 參數是 *debug.BuildInfo，而不是在函式裡呼叫 [debug.ReadBuildInfo]：讀的事交給呼叫端（main），
// 測試才能傳假的；bi 為 nil 代表執行檔沒有建置資訊。
func From(bi *debug.BuildInfo, injected string) Info {
	info := Info{Version: injected}
	if bi != nil {
		info.GoVersion = bi.GoVersion
		if info.Version == "" {
			info.Version = bi.Main.Version
		}
		for _, s := range bi.Settings {
			switch s.Key {
			case "vcs.revision":
				info.Revision = s.Value
			case "vcs.time":
				info.Time = s.Value
			case "vcs.modified":
				info.Modified = s.Value == "true"
			}
		}
	}
	if info.Version == "" {
		info.Version = "unknown"
	}
	return info
}

// String 印成一行：版本、commit 前 12 碼（有未提交的修改時加 -dirty）、commit 時間、Go 版本。
// 沒有 VCS 資訊（容器的建置脈絡沒帶 .git、go run）時，commit 的位置寫 "(no vcs info)"。
func (i Info) String() string {
	parts := []string{i.Version}
	if i.Revision == "" {
		parts = append(parts, "(no vcs info)")
	} else {
		rev := i.Revision[:min(len(i.Revision), 12)]
		if i.Modified {
			rev += "-dirty"
		}
		parts = append(parts, rev, i.Time)
	}
	if i.GoVersion != "" {
		parts = append(parts, i.GoVersion)
	}
	return strings.Join(parts, " ")
}

// ANCHOR_END: info
