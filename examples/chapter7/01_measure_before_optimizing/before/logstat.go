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

// entry 是一行日誌解析後的結果。
type entry struct {
	id, path   string
	status, ms int
}

// ANCHOR: parsefast

// parseFast 是手寫的解析：不用 strings.Split 與 strconv，逐字元掃過一行。
func parseFast(line string) (e entry, ok bool) {
	var start, end [5]int
	n, from := 0, 0
	for i := 0; i <= len(line); i++ {
		if i < len(line) && line[i] != ' ' {
			continue
		}
		if n == 5 {
			return e, false
		}
		start[n], end[n] = from, i
		n++
		from = i + 1
	}
	if n != 5 {
		return e, false
	}
	status, ok1 := atoiFast(line[start[3]:end[3]])
	ms, ok2 := atoiFast(line[start[4]:end[4]])
	return entry{line[start[0]:end[0]], line[start[2]:end[2]], status, ms}, ok1 && ok2
}

// atoiFast 手寫的整數解析，只接受 1 到 9 位的數字。
func atoiFast(s string) (int, bool) {
	if len(s) == 0 || len(s) > 9 {
		return 0, false
	}
	n := 0
	for i := 0; i < len(s); i++ {
		d := s[i] - '0'
		if d > 9 {
			return 0, false
		}
		n = n*10 + int(d)
	}
	return n, true
}

// ANCHOR_END: parsefast

// parseSimple 用標準庫解析，規則與 parseFast 相同。
func parseSimple(line string) (entry, bool) {
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

// atoi 只接受 1 到 9 位的數字。
func atoi(s string) (int, bool) {
	n, err := strconv.ParseUint(s, 10, 32)
	return int(n), err == nil && len(s) <= 9
}

// idSegment 比對路徑裡的數字片段。
var idSegment = regexp.MustCompile(`/[0-9]+\b`)

// normalize 把路徑裡的數字片段換成 :id，讓 /users/42 與 /users/7 算同一條路由。
func normalize(path string) string {
	return idSegment.ReplaceAllString(path, "/:id")
}

func summarize(lines []string, parse func(string) (entry, bool)) []RouteStat {
	byRoute := map[string]*RouteStat{}
	var seen []string // 已經算過的請求編號
	for _, line := range lines {
		e, ok := parse(line)
		if !ok {
			continue // 壞掉的行直接略過
		}
		if slices.Contains(seen, e.id) {
			continue // 重送的行
		}
		seen = append(seen, e.id)
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

// Summarize 彙總一批日誌，結果依路由排序；格式不對的行會被略過。
// 解析是「最佳化」過的手寫版本。
func Summarize(lines []string) []RouteStat { return summarize(lines, parseFast) }

// SummarizeSimple 與 Summarize 的結果相同，但用標準庫解析，
// 也就是「最佳化」之前的寫法；留在這裡是為了拿來比較。
func SummarizeSimple(lines []string) []RouteStat { return summarize(lines, parseSimple) }
