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
func ParseLine(line string) (*Entry, error) {
	method, rest, ok := strings.Cut(line, " ")
	if !ok {
		return nil, errors.New("missing path")
	}
	path, status, ok := strings.Cut(rest, " ")
	if !ok {
		return nil, errors.New("missing status")
	}
	code, err := strconv.Atoi(status)
	if err != nil || code < 100 || code > 599 {
		return nil, fmt.Errorf("bad status %q", status)
	}
	return &Entry{Method: method, Path: path, Status: code}, nil
}

// ANCHOR_END: parse

// ANCHOR: route

// Route 回傳 "GET /users/42 5xx" 這種格式的路由鍵，狀態碼只留類別。
func Route(line string) (string, error) {
	e, err := ParseLine(line)
	if err != nil {
		return "", err
	}
	buf := make([]byte, 0, len(e.Method)+len(e.Path)+6)
	buf = append(buf, e.Method...)
	buf = append(buf, ' ')
	buf = append(buf, e.Path...)
	buf = append(buf, ' ', byte('0'+e.Status/100), 'x', 'x')
	return string(buf), nil
}

// ANCHOR_END: route
