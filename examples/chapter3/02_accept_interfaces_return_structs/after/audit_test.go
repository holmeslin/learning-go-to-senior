package audit_test

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	audit "github.com/holmeslin/learning-go-to-senior/examples/chapter3/02_accept_interfaces_return_structs/after"
)

// ANCHOR: test

// spyWriter 是自己寫的 io.Writer，把收到的內容存下來。
type spyWriter struct{ got []byte }

func (s *spyWriter) Write(p []byte) (int, error) {
	s.got = append(s.got, p...)
	return len(p), nil
}

func TestNewAcceptsAnyWriter(t *testing.T) {
	var buf bytes.Buffer
	var sb strings.Builder
	var spy spyWriter

	tests := []struct {
		name string
		w    io.Writer
		got  func() string
	}{
		{"bytes.Buffer", &buf, buf.String},
		{"strings.Builder", &sb, sb.String},
		{"custom writer", &spy, func() string { return string(spy.got) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := audit.New(tt.w)
			l.Log("login")
			if err := l.Flush(); err != nil {
				t.Fatalf("Flush() error = %v", err)
			}
			if want := "login\n"; tt.got() != want {
				t.Errorf("writer got %q; want %q", tt.got(), want)
			}
		})
	}
}

// ANCHOR_END: test

func TestLogIsBufferedUntilFlush(t *testing.T) {
	var buf bytes.Buffer
	l := audit.New(&buf)

	l.Log("login")
	if buf.Len() != 0 {
		t.Errorf("before Flush, writer got %q; want nothing", buf.String())
	}
	if err := l.Flush(); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}
	if want := "login\n"; buf.String() != want {
		t.Errorf("after Flush, writer got %q; want %q", buf.String(), want)
	}
}

var errDisk = errors.New("disk full")

// failWriter 的每次寫入都失敗。
type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, errDisk }

func TestFlushReportsWriteError(t *testing.T) {
	l := audit.New(failWriter{})
	l.Log("login")

	if err := l.Flush(); !errors.Is(err, errDisk) {
		t.Errorf("Flush() error = %v; want %v", err, errDisk)
	}
}
