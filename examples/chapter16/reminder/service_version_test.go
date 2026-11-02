package reminder_test

import (
	"context"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter16/reminder"
	"github.com/holmeslin/learning-go-to-senior/examples/chapter16/reminder/internal/buildinfo"
)

// ANCHOR: version

// 版本同時出現在啟動日誌與診斷端點的 /version：值班的人從日誌或 curl 都問得到「現在跑的是哪個建置」。
// 這個測試不在 bubble 裡（要真的 socket）；開關關閉，不需要供應商。
func TestServiceReportsItsVersionInTheStartupLogAndTheDiagnosticEndpoint(t *testing.T) {
	h := newHarness(t, map[string]string{"REMINDER_SEND_ENABLED": "false"})
	h.cfg.DiagAddr = "127.0.0.1:0"
	sink := logSink{records: make(chan slog.Record, 64)}
	build := buildinfo.Info{Version: "v1.5.0", Revision: "7a0c240b07b41e3a", Time: "2030-06-10T08:00:00Z", GoVersion: "go1.27.0"}
	svc := &reminder.Service{Config: h.cfg, Logger: slog.New(sink), Build: build}
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- svc.Run(ctx) }()

	started := sink.await(t, "diagnostic server listening", "reminder service started")
	if got, want := started["reminder service started"]["version"], build.String(); got != want {
		t.Errorf("startup log version = %q, want %q", got, want)
	}
	url := "http://" + started["diagnostic server listening"]["addr"] + "/version"
	const want = `{"version":"v1.5.0","revision":"7a0c240b07b41e3a","time":"2030-06-10T08:00:00Z","modified":false,"go_version":"go1.27.0"}` + "\n"
	if code, body := get(t, url); code != http.StatusOK || body != want {
		t.Errorf("GET /version = %d %q, want 200 %q", code, body, want)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Run() error = %v, want nil", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Run() did not return after the signal")
	}
}

// ANCHOR_END: version
