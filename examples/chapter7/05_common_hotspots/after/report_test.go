package accesslog_test

import (
	"testing"

	accesslog "github.com/holmeslin/learning-go-to-senior/examples/chapter7/05_common_hotspots/after"
	old "github.com/holmeslin/learning-go-to-senior/examples/chapter7/05_common_hotspots/before"
)

// ANCHOR: allocs

// sample 產生 n 筆紀錄，每四筆有一筆是失敗請求，路徑都已經是小寫。
func sample(n int) []accesslog.Entry {
	entries := make([]accesslog.Entry, n)
	for i := range entries {
		status := 200
		if i%4 == 0 {
			status = 404
		}
		entries[i] = accesslog.Entry{Method: "GET", Path: "/api/users?page=2", Status: status}
	}
	return entries
}

// 配置次數與紀錄筆數無關：一份失敗清單，加上一個放得下整份報告的緩衝區。
func TestReportAllocs(t *testing.T) {
	entries := sample(1000)
	got := testing.AllocsPerRun(10, func() { _ = accesslog.Report(entries) })
	if want := 2.0; got > want {
		t.Errorf("Report(1000 entries) allocs = %v, want <= %v", got, want)
	}
}

// ANCHOR_END: allocs

// 改寫前後的結果必須一致，包含大小寫、查詢字串與不該出現的成功請求。
func TestReportMatchesOld(t *testing.T) {
	tests := [][]accesslog.Entry{
		nil,
		sample(50),
		{
			{Method: "POST", Path: "/API/Orders?id=7", Status: 500},
			{Method: "GET", Path: "/ok", Status: 200},
			{Method: "GET", Path: "/Login", Status: 401},
			{Method: "GET", Path: "/ÄÖ?x=1", Status: 399},
			{Method: "DELETE", Path: "", Status: 400},
		},
	}
	for _, entries := range tests {
		oldEntries := make([]old.Entry, len(entries))
		for i, e := range entries {
			oldEntries[i] = old.Entry(e)
		}
		if got, want := accesslog.Report(entries), old.Report(oldEntries); got != want {
			t.Errorf("Report(%v) = %q, want %q", entries, got, want)
		}
	}
}
