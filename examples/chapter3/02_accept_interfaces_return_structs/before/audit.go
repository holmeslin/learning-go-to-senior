// Package audit 把稽核事件寫到檔案。
package audit

import (
	"bufio"
	"fmt"
	"os"
)

// ANCHOR: logger

// Logger 記錄稽核事件。
type Logger interface {
	Log(event string)
}

// New 回傳寫入 f 的 Logger。f 為 nil 表示不記錄，回傳 nil。
func New(f *os.File) Logger {
	var l *bufLogger
	if f != nil {
		l = &bufLogger{w: bufio.NewWriter(f)}
	}
	return l
}

type bufLogger struct {
	w *bufio.Writer
}

func (l *bufLogger) Log(event string) {
	fmt.Fprintln(l.w, event)
}

// Flush 把緩衝區裡的紀錄送出，並回傳寫入過程中遇到的錯誤。
func (l *bufLogger) Flush() error {
	return l.w.Flush()
}

// ANCHOR_END: logger
