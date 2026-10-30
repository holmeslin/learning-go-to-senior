package graceful_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"

	graceful "github.com/holmeslin/learning-go-to-senior/examples/chapter10/06_graceful_shutdown/before"
)

// ANCHOR: helpers

// slowServer 建立一個 /slow 會卡在 gate 上的伺服器，started 在 handler 開始處理時關閉。
// 只借 httptest 的 Listener 與 Config：由 Exit 或 ShutdownOnly 來 Serve。
func slowServer(gate chan struct{}) (ts *httptest.Server, started chan struct{}) {
	started = make(chan struct{})
	ts = httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-gate
		fmt.Fprint(w, "saved")
	}))
	return ts, started
}

// getSlow 在背景發出 GET /slow，結果（回應內容或錯誤）送進回傳的 channel。
func getSlow(ts *httptest.Server) <-chan string {
	result := make(chan string, 1)
	go func() {
		resp, err := http.Get("http://" + ts.Listener.Addr().String() + "/slow")
		if err != nil {
			result <- "error"
			return
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		result <- string(b)
	}()
	return result
}

// ANCHOR_END: helpers

// ANCHOR: exit

func Example_exit() {
	gate, workGate := make(chan struct{}), make(chan struct{})
	ts, started := slowServer(gate)
	var jobStarted, jobFinished atomic.Bool
	jobRunning := make(chan struct{})
	work := func(ctx context.Context) {
		jobStarted.Store(true)
		close(jobRunning)
		<-workGate // 一筆要很久的工作
		jobFinished.Store(true)
	}

	ctx, cancel := context.WithCancel(context.Background())
	exited := make(chan error, 1)
	go func() { exited <- graceful.Exit(ctx, ts.Config, ts.Listener, work) }()
	response := getSlow(ts)
	<-started
	<-jobRunning

	cancel()          // 收到 SIGTERM
	<-exited          // Exit 回傳，main 接著結束行程
	ts.Config.Close() // 行程結束時，作業系統會關掉所有連線；測試裡用 Close 代替
	fmt.Println("request:", <-response)
	fmt.Println("job started:", jobStarted.Load(), "finished:", jobFinished.Load())

	close(gate) // 只是讓測試裡的 goroutine 離開
	close(workGate)
	// Output:
	// request: error
	// job started: true finished: false
}

// ANCHOR_END: exit

// ANCHOR: shutdown

func Example_shutdownOnly() {
	gate, workGate := make(chan struct{}), make(chan struct{})
	ts, started := slowServer(gate)
	var jobFinished atomic.Bool
	jobRunning := make(chan struct{})
	work := func(ctx context.Context) {
		close(jobRunning)
		<-workGate
		jobFinished.Store(true)
	}
	shutdownBegan := make(chan struct{})
	ts.Config.RegisterOnShutdown(func() { close(shutdownBegan) })

	ctx, cancel := context.WithCancel(context.Background())
	exited := make(chan error, 1)
	go func() { exited <- graceful.ShutdownOnly(ctx, ts.Config, ts.Listener, work) }()
	response := getSlow(ts)
	<-started
	<-jobRunning

	cancel()
	<-shutdownBegan
	close(gate) // 請求在 Shutdown 開始之後才做完
	fmt.Println("request:", <-response)
	fmt.Println("shutdown:", <-exited) // main 在這裡結束行程
	fmt.Println("job finished:", jobFinished.Load())

	close(workGate)
	// Output:
	// request: saved
	// shutdown: <nil>
	// job finished: false
}

// ANCHOR_END: shutdown
