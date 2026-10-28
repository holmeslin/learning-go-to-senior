// Package accesslog 整理存取日誌：產生失敗請求的報告，並統計各路由的請求數。
package accesslog

import (
	"strconv"
	"strings"
)

// Entry 是一筆請求紀錄。
type Entry struct {
	Method string // HTTP 方法，例如 GET
	Path   string // 請求路徑，可含查詢字串
	Status int    // 回應的狀態碼
}

// ANCHOR: report

// Report 回傳狀態碼 400 以上的請求，每筆一行：「方法 路由 狀態碼」。
// 路由是小寫、去掉查詢字串的路徑。
func Report(entries []Entry) string {
	failed := make([]Entry, 0, len(entries))
	for _, e := range entries {
		if e.Status >= 400 {
			failed = append(failed, e)
		}
	}

	var b strings.Builder
	b.Grow(len(failed) * 48) // 估計每行 48 bytes；估少了 Builder 會自己擴充
	var num [20]byte
	for _, e := range failed {
		b.WriteString(e.Method)
		b.WriteByte(' ')
		b.WriteString(route(e.Path))
		b.WriteByte(' ')
		b.Write(strconv.AppendInt(num[:0], int64(e.Status), 10))
		b.WriteByte('\n')
	}
	return b.String()
}

// route 去掉查詢字串並轉成小寫；路徑已經是小寫時不會配置。
func route(path string) string {
	path, _, _ = strings.Cut(path, "?")
	return strings.ToLower(path)
}

// ANCHOR_END: report
