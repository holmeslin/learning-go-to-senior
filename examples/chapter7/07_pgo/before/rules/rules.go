// Package rules 統計存取紀錄裡符合規則的筆數。
package rules

import (
	"fmt"
	"strings"
)

// Request 是一筆已解析的存取紀錄。
type Request struct {
	// Path 是請求的路徑。
	Path string
	// Status 是回應的狀態碼。
	Status int
	// Bytes 是回應的大小。
	Bytes int
}

// Stats 是一批紀錄的統計。
type Stats struct {
	// Total 是紀錄總數。
	Total int
	// Matched 是符合規則的筆數。
	Matched int
	// Bytes 是符合規則的回應大小總和。
	Bytes int
}

// Rule 判斷一筆紀錄要不要計入。
type Rule interface {
	// Match 回報 r 是否符合規則。
	Match(r *Request) bool
}

// PathPrefix 比對路徑的前綴，忽略大小寫。
type PathPrefix struct {
	// Prefix 是要比對的前綴。
	Prefix string
}

// Match 實作 Rule。
func (p *PathPrefix) Match(r *Request) bool {
	if len(r.Path) < len(p.Prefix) {
		return false
	}
	for i := 0; i < len(p.Prefix); i++ {
		a, b := r.Path[i], p.Prefix[i]
		if 'A' <= a && a <= 'Z' {
			a += 'a' - 'A'
		}
		if 'A' <= b && b <= 'Z' {
			b += 'a' - 'A'
		}
		if a != b {
			return false
		}
	}
	return true
}

// ServerError 比對 5xx 的回應。
type ServerError struct{}

// Match 實作 Rule。
func (*ServerError) Match(r *Request) bool { return r.Status >= 500 }

// ANCHOR: tally

// Tally 統計 reqs 裡符合 rule 的紀錄。
//
// shortcut: 標 noinline，不讓 benchmark 的迴圈把整個函式 inline 進去；
// 那樣量到的不再是實際執行檔裡 main 只呼叫一次的 Tally。
//
//go:noinline
func Tally(reqs []Request, rule Rule) Stats {
	st := Stats{Total: len(reqs)}
	for i := range reqs {
		if rule.Match(&reqs[i]) {
			st.Matched++
			st.Bytes += reqs[i].Bytes
		}
	}
	return st
}

// ANCHOR_END: tally

// ParseRule 把 "5xx" 或 "path:/api/" 這樣的設定轉成 Rule。
func ParseRule(spec string) (Rule, error) {
	if spec == "5xx" {
		return &ServerError{}, nil
	}
	if prefix, ok := strings.CutPrefix(spec, "path:"); ok {
		return &PathPrefix{Prefix: prefix}, nil
	}
	return nil, fmt.Errorf("unknown rule %q", spec)
}

// Sample 產生 n 筆固定內容的紀錄：每 10 筆有 7 筆是 /api/ 開頭，每 20 筆有 1 筆是 5xx。
func Sample(n int) []Request {
	paths := []string{"/API/users", "/api/orders", "/static/app.js", "/api/items", "/healthz", "/Api/search", "/api/v2/items", "/index.html", "/api/login", "/api/Cart"}
	reqs := make([]Request, n)
	for i := range reqs {
		status := 200
		if i%20 == 7 {
			status = 503
		}
		reqs[i] = Request{Path: paths[i%len(paths)], Status: status, Bytes: 100 + i%50}
	}
	return reqs
}
