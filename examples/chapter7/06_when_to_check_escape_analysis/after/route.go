// Package accesslog 把存取日誌的一行整理成指標用的路由鍵。
package accesslog

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Entry 是一行存取日誌解析後的欄位。
type Entry struct {
	// Method 是 HTTP 方法。
	Method string
	// Path 是請求的路徑。
	Path string
	// Status 是回應的狀態碼。
	Status int
}

// ANCHOR: parse

// ParseLine 解析 "GET /users/42 200" 這種格式的一行日誌。
func ParseLine(line string) (Entry, error) {
	method, rest, ok := strings.Cut(line, " ")
	if !ok {
		return Entry{}, errors.New("missing path")
	}
	path, status, ok := strings.Cut(rest, " ")
	if !ok {
		return Entry{}, errors.New("missing status")
	}
	code, err := strconv.Atoi(status)
	if err != nil || code < 100 || code > 599 {
		return Entry{}, fmt.Errorf("bad status %q", status)
	}
	return Entry{Method: method, Path: path, Status: code}, nil
}

// ANCHOR_END: parse

// ANCHOR: route

// AppendRoute 把 line 的路由鍵（例如 "GET /users/42 5xx"，狀態碼只留類別）
// 附加到 dst 後面並回傳。line 無法解析時，原樣回傳 dst 與錯誤。
func AppendRoute(dst []byte, line string) ([]byte, error) {
	e, err := ParseLine(line)
	if err != nil {
		return dst, err
	}
	dst = append(dst, e.Method...)
	dst = append(dst, ' ')
	dst = append(dst, e.Path...)
	return append(dst, ' ', byte('0'+e.Status/100), 'x', 'x'), nil
}

// Route 回傳 line 的路由鍵，不在乎配置的呼叫端用這個就好。
func Route(line string) (string, error) {
	b, err := AppendRoute(nil, line)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ANCHOR_END: route
