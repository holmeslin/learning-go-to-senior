package graceful_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"

	graceful "github.com/holmeslin/learning-go-to-senior/examples/chapter10/06_graceful_shutdown/after"
)

// ANCHOR: example

func Example() {
	var mu sync.Mutex
	var events []string
	note := func(s string) { mu.Lock(); events = append(events, s); mu.Unlock() }

	gate, inHandler := make(chan struct{}), make(chan struct{})
	svc := &graceful.Service{
		Requests: time.Minute, Workers: time.Minute, Grace: time.Minute,
		Work: func(ctx context.Context) { <-ctx.Done(); note("worker stopping") },
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /readyz", svc.Ready)
	mux.HandleFunc("GET /slow", func(w http.ResponseWriter, r *http.Request) {
		close(inHandler)
		<-gate
		note("request finished")
		fmt.Fprint(w, "saved")
	})
	ts := httptest.NewUnstartedServer(mux) // 只借它的 Listener 與 Config，由 Run 來 Serve
	svc.Server = ts.Config
	shutdownBegan := make(chan struct{})
	ts.Config.RegisterOnShutdown(func() { close(shutdownBegan) })

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- svc.Run(ctx, ts.Listener) }()

	body := make(chan string, 1)
	go func() {
		resp, err := http.Get("http://" + ts.Listener.Addr().String() + "/slow")
		if err != nil {
			body <- err.Error()
			return
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		body <- string(b)
	}()

	<-inHandler
	cancel() // 收到結束訊號
	<-shutdownBegan
	close(gate) // Shutdown 開始之後，進行中的請求才做完
	fmt.Println("response:", <-body)
	fmt.Println("run:", <-done)
	mu.Lock()
	fmt.Println(strings.Join(events, ", "))
	mu.Unlock()
	// Output:
	// response: saved
	// run: <nil>
	// request finished, worker stopping
}

// ANCHOR_END: example
