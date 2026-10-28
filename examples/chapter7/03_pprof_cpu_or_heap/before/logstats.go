// Package logstats 把一批 access log 彙總成每個路徑的請求數、錯誤數與 P95 延遲。
package logstats

import (
	"bytes"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Stat 是一個路徑的彙總。
type Stat struct {
	Count  int       // 請求數
	Errors int       // 狀態碼 5xx 的請求數
	P95    int       // 第 95 百分位延遲，單位毫秒
	Last   time.Time // 最後一筆請求的時間
}

var errMalformed = errors.New("malformed line")

type entry struct {
	at     time.Time
	path   string
	status int
	ms     int
}

type group struct {
	count, errors int
	last          time.Time
	ms            []int
}

// ANCHOR: summarize

// Summarize 彙總 lines。每行的格式是「時間 方法 路徑 狀態碼 延遲毫秒」，
// 路徑裡純數字的片段會換成 :id（/api/users/42 彙總到 /api/users/:id）。
// 任何一行格式不對就回傳錯誤，訊息帶行號。
func Summarize(lines []string) (map[string]Stat, error) {
	groups := make(map[string]*group)
	for i, line := range lines {
		e, err := parseLine(line)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", i+1, err)
		}
		key := normalize(e.path)
		g := groups[key]
		if g == nil {
			g = new(group)
			groups[key] = g
		}
		g.count++
		if e.status >= 500 {
			g.errors++
		}
		g.ms = append(g.ms, e.ms)
		g.last = later(g.last, e.at)
	}
	return finish(groups), nil
}

// maxPathLen 是預留給路徑的長度，預留夠大就不必在寫入時擴張。
const maxPathLen = 4096

func normalize(path string) string {
	var b bytes.Buffer
	b.Grow(maxPathLen)
	first := true
	for seg := range strings.SplitSeq(path, "/") {
		if !first {
			b.WriteByte('/')
		}
		first = false
		if isNumber(seg) {
			seg = ":id"
		}
		b.WriteString(seg)
	}
	return b.String()
}

// ANCHOR_END: summarize

func parseLine(line string) (entry, error) {
	ts, rest, ok1 := strings.Cut(line, " ")
	_, rest, ok2 := strings.Cut(rest, " ") // 方法目前用不到
	path, rest, ok3 := strings.Cut(rest, " ")
	status, ms, ok4 := strings.Cut(rest, " ")
	if !ok1 || !ok2 || !ok3 || !ok4 {
		return entry{}, errMalformed
	}
	at, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return entry{}, err
	}
	code, err := strconv.Atoi(status)
	if err != nil {
		return entry{}, err
	}
	d, err := strconv.Atoi(ms)
	if err != nil {
		return entry{}, err
	}
	return entry{at: at, path: path, status: code, ms: d}, nil
}

func isNumber(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func later(a, b time.Time) time.Time {
	if b.After(a) {
		return b
	}
	return a
}

func finish(groups map[string]*group) map[string]Stat {
	stats := make(map[string]Stat, len(groups))
	for key, g := range groups {
		slices.Sort(g.ms)
		p95 := g.ms[(len(g.ms)*95+99)/100-1] // nearest-rank
		stats[key] = Stat{Count: g.count, Errors: g.errors, P95: p95, Last: g.last}
	}
	return stats
}
