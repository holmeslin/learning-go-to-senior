// Package version 把 Go 記在執行檔裡的建置資訊整理成可以印出來的版本資訊。
package version

import (
	"encoding/json"
	"net/http"
	"runtime/debug"
	"strings"
)

// ANCHOR: info

// Info 是一個執行檔的版本資訊，由 [FromBuildInfo] 整理而來。
type Info struct {
	// Version 是主模組的版本；go run 與測試執行檔是 "(devel)"。
	Version string `json:"version"`
	// Revision 是建置時的 commit 雜湊（vcs.revision），沒有 VCS 資訊時為空。
	Revision string `json:"revision,omitempty"`
	// Time 是該 commit 的時間（vcs.time，RFC 3339），沒有 VCS 資訊時為空。
	Time string `json:"time,omitempty"`
	// Modified 表示建置時工作目錄有未提交的修改（vcs.modified）。
	Modified bool `json:"modified"`
	// GoVersion 是建置用的 Go 版本。
	GoVersion string `json:"go_version,omitempty"`
	// Build 是只有建置系統才知道的值（例如 CI 的執行編號），由 -ldflags -X 注入。
	Build string `json:"build,omitempty"`
}

// FromBuildInfo 把 bi 整理成 [Info]；build 是用 -X 注入的值，沒有就傳空字串。
// 參數是 *debug.BuildInfo，而不是在函式裡呼叫 [debug.ReadBuildInfo]，
// 呼叫端（main）負責讀，測試才能傳假的。bi 為 nil（執行檔沒有建置資訊）時版本是 "unknown"。
func FromBuildInfo(bi *debug.BuildInfo, build string) Info {
	if bi == nil {
		return Info{Version: "unknown", Build: build}
	}
	info := Info{Version: bi.Main.Version, GoVersion: bi.GoVersion, Build: build}
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
	return info
}

// String 印成一行：版本、commit 前 12 碼（有未提交的修改時加 -dirty）、commit 時間、Go 版本，
// 有 Build 時最後再加上。
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
	if i.Build != "" {
		parts = append(parts, "build", i.Build)
	}
	return strings.Join(parts, " ")
}

// Handler 回傳版本端點的 handler，以 JSON 回應 i。
func Handler(i Info) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(i) // 寫入失敗代表客戶端已斷線，沒有人可以回報
	})
}

// ANCHOR_END: info
