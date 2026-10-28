// Package accesslog 整理存取日誌：產生失敗請求的報告，並統計各路由的請求數。
package accesslog

import (
	"bytes"
	"strconv"
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
	var failed []Entry
	for _, e := range entries {
		if e.Status >= 400 {
			failed = append(failed, e)
		}
	}

	out := ""
	for _, e := range failed {
		out += e.Method + " " + route(e.Path) + " " + strconv.Itoa(e.Status) + "\n"
	}
	return out
}

// route 去掉查詢字串並轉成小寫。
func route(path string) string {
	b := []byte(path)
	b = bytes.ToLower(b)
	if i := bytes.IndexByte(b, '?'); i >= 0 {
		b = b[:i]
	}
	return string(b)
}

// ANCHOR_END: report
