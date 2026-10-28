// Package logstat 把一批存取日誌彙總成每條路由的統計。
// 日誌格式是以單一空白分隔的五個欄位：請求編號、方法、路徑、狀態碼、毫秒數。
// log shipper 重送時，同一個請求編號會出現不只一次，只算第一次。
package logstat

import (
	"cmp"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// RouteStat 是一條路由的統計。路徑裡的數字片段已經換成 :id。
type RouteStat struct {
	Route   string // 正規化後的路徑，例如 /users/:id
	Count   int    // 請求數
	Errors  int    // 狀態碼 500 以上的請求數
	TotalMS int    // 毫秒數的總和
}

// idSegment 比對路徑裡的數字片段。
var idSegment = regexp.MustCompile(`/[0-9]+\b`)

// normalize 把路徑裡的數字片段換成 :id，讓 /users/42 與 /users/7 算同一條路由。
func normalize(path string) string {
	return idSegment.ReplaceAllString(path, "/:id")
}

// ANCHOR: summarize

// entry 是一行日誌解析後的結果。
type entry struct {
	id, path   string
	status, ms int
}

// parse 解析一行；格式不對時回傳 false。
func parse(line string) (entry, bool) {
	if strings.Count(line, " ") != 4 {
		return entry{}, false
	}
	id, rest, _ := strings.Cut(line, " ")
	_, rest, _ = strings.Cut(rest, " ") // 方法用不到
	path, rest, _ := strings.Cut(rest, " ")
	statusText, msText, _ := strings.Cut(rest, " ")
	status, ok1 := atoi(statusText)
	ms, ok2 := atoi(msText)
	return entry{id, path, status, ms}, ok1 && ok2
}

// Summarize 彙總一批日誌，結果依路由排序；格式不對的行會被略過。
func Summarize(lines []string) []RouteStat {
	byRoute := map[string]*RouteStat{}
	seen := map[string]struct{}{} // 已經算過的請求編號
	for _, line := range lines {
		e, ok := parse(line)
		if !ok {
			continue // 壞掉的行直接略過
		}
		if _, dup := seen[e.id]; dup {
			continue // 重送的行
		}
		seen[e.id] = struct{}{}
		route := normalize(e.path)
		s := byRoute[route]
		if s == nil {
			s = &RouteStat{Route: route}
			byRoute[route] = s
		}
		s.Count++
		if e.status >= 500 {
			s.Errors++
		}
		s.TotalMS += e.ms
	}
	out := make([]RouteStat, 0, len(byRoute))
	for _, s := range byRoute {
		out = append(out, *s)
	}
	slices.SortFunc(out, func(a, b RouteStat) int { return cmp.Compare(a.Route, b.Route) })
	return out
}

// ANCHOR_END: summarize

// atoi 只接受 1 到 9 位的數字。
func atoi(s string) (int, bool) {
	n, err := strconv.ParseUint(s, 10, 32)
	return int(n), err == nil && len(s) <= 9
}
