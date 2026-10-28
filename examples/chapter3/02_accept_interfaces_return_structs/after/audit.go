// Package audit 把稽核事件寫到 io.Writer。
package audit

import (
	"bufio"
	"fmt"
	"io"
)

// ANCHOR: logger

// Logger 記錄稽核事件。寫入先經過緩衝，呼叫 Flush 才會真的送出。
type Logger struct {
	w *bufio.Writer
}

// New 回傳寫入 w 的 Logger。w 不可為 nil；不想記錄時傳 io.Discard。
func New(w io.Writer) *Logger {
	return &Logger{w: bufio.NewWriter(w)}
}

// Log 記錄一筆事件。寫入的錯誤會留到 Flush 才回報。
func (l *Logger) Log(event string) {
	fmt.Fprintln(l.w, event)
}

// Flush 把緩衝區裡的紀錄送出，並回傳寫入過程中遇到的錯誤。
func (l *Logger) Flush() error {
	return l.w.Flush()
}

// ANCHOR_END: logger
