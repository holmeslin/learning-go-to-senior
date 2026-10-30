package graceful_test

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	graceful "github.com/holmeslin/learning-go-to-senior/examples/chapter10/06_graceful_shutdown/after"
)

// ANCHOR: net

// memListener 是 synctest 用的假網路：連線是 net.Pipe，不經過 socket。
// bubble 裡的 socket I/O 不算 durably blocked，假時間不會前進；Pipe 是 channel，會。
type memListener struct {
	conns chan net.Conn
	done  chan struct{}
	once  sync.Once
}

func (l *memListener) Accept() (net.Conn, error) {
	select {
	case c := <-l.conns:
		return c, nil
	case <-l.done:
		return nil, net.ErrClosed
	}
}

func (l *memListener) Close() error   { l.once.Do(func() { close(l.done) }); return nil }
func (l *memListener) Addr() net.Addr { return &net.UnixAddr{Name: "mem", Net: "mem"} }

// dial 把一端的 Pipe 交給 Accept；listener 關了就拒絕連線。
func (l *memListener) dial(ctx context.Context, _, _ string) (net.Conn, error) {
	client, server := net.Pipe()
	select {
	case l.conns <- server:
		return client, nil
	case <-l.done:
		return nil, errors.New("connection refused")
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// ANCHOR_END: net

// ANCHOR: launch

// launch 在 bubble 裡啟動 svc：/slow 卡在 gate 上，/fast 立刻回應。
// 回傳的 cancel 代表「收到結束訊號」，done 收到 Run 的結果。
func launch(t *testing.T, svc *graceful.Service, gate chan struct{}) (client *http.Client, cancel context.CancelFunc, done <-chan error) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /readyz", svc.Ready)
	mux.HandleFunc("GET /fast", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })
	mux.HandleFunc("GET /slow", func(w http.ResponseWriter, r *http.Request) { <-gate; w.Write([]byte("saved")) })
	svc.Server = &http.Server{Handler: mux}

	ln := &memListener{conns: make(chan net.Conn), done: make(chan struct{})}
	ctx, cancel := context.WithCancel(t.Context())
	result := make(chan error, 1)
	go func() { result <- svc.Run(ctx, ln) }()

	client = &http.Client{Transport: &http.Transport{DialContext: ln.dial, DisableKeepAlives: true}}
	return client, cancel, result
}

// get 回傳狀態碼；連不上或連線被切斷時回傳錯誤。
func get(c *http.Client, path string) (int, error) {
	resp, err := c.Get("http://service" + path)
	if err != nil {
		return 0, err
	}
	resp.Body.Close()
	return resp.StatusCode, nil
}

// ANCHOR_END: launch

// ANCHOR: order

func TestShutdownOrder(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		gate, stopWork := make(chan struct{}), make(chan struct{})
		stopping := make(chan struct{})
		svc := &graceful.Service{
			Drain: 5 * time.Second, Requests: 10 * time.Second, Workers: 10 * time.Second, Grace: 30 * time.Second,
			Work: func(ctx context.Context) {
				<-ctx.Done()
				close(stopping)
				<-stopWork // 收尾要一點時間
			},
		}
		client, cancel, done := launch(t, svc, gate)

		inflight := make(chan int, 1)
		go func() { code, _ := get(client, "/slow"); inflight <- code }()
		synctest.Wait() // 請求已進到 handler
		begin := time.Now()
		cancel()
		synctest.Wait() // Run 正在等 Drain

		// 第一步：readiness 失敗，但還在接新請求。
		if code, _ := get(client, "/readyz"); code != http.StatusServiceUnavailable {
			t.Errorf("GET /readyz during drain = %d, want 503", code)
		}
		if code, err := get(client, "/fast"); code != http.StatusOK {
			t.Errorf("GET /fast during drain = %d, %v, want 200", code, err)
		}

		// 第二步：Drain 結束，停止接受新連線，進行中的請求還沒被打斷。
		time.Sleep(5 * time.Second)
		synctest.Wait()
		if _, err := get(client, "/fast"); err == nil {
			t.Error("new request was accepted after Shutdown began")
		}
		select {
		case <-inflight:
			t.Fatal("in-flight request ended before its handler returned")
		default:
		}
		select {
		case <-stopping:
			t.Error("worker was told to stop while a request was still running")
		default:
		}

		// 第三步：請求做完之後才通知背景工作，Run 要等到它收尾。
		close(gate)
		if code := <-inflight; code != http.StatusOK {
			t.Errorf("in-flight GET /slow = %d, want 200", code)
		}
		<-stopping // Shutdown 回傳之後，Run 才通知背景工作
		select {
		case err := <-done:
			t.Fatalf("Run returned (%v) before the worker finished", err)
		default:
		}
		close(stopWork)
		if err := <-done; err != nil {
			t.Errorf("Run() = %v, want nil", err)
		}
		// Shutdown 每隔幾毫秒輪詢一次閒置連線，所以會比 Drain 多一點點，但遠低於 Requests 的上限。
		if got := time.Since(begin); got < 5*time.Second || got > 6*time.Second {
			t.Errorf("shutdown took %v, want about 5s (only the drain wait)", got)
		}
	})
}

// ANCHOR_END: order

// ANCHOR: stuck

func TestStuckRequestIsCutAtLimit(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		gate := make(chan struct{})
		defer close(gate) // bubble 結束前，卡住的 handler 必須能離開
		svc := &graceful.Service{
			Drain: 5 * time.Second, Requests: 10 * time.Second, Workers: 10 * time.Second, Grace: 30 * time.Second,
			Work: func(ctx context.Context) { <-ctx.Done() },
		}
		client, cancel, done := launch(t, svc, gate)

		inflight := make(chan error, 1)
		go func() { _, err := get(client, "/slow"); inflight <- err }()
		synctest.Wait()
		begin := time.Now()
		cancel()

		if err := <-done; !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("Run() = %v, want context.DeadlineExceeded", err)
		}
		if got := time.Since(begin); got != 15*time.Second {
			t.Errorf("shutdown took %v, want 15s (drain 5s + requests 10s)", got)
		}
		if err := <-inflight; err == nil {
			t.Error("stuck request got a response, want the connection cut")
		}
	})
}

// ANCHOR_END: stuck

// ANCHOR: grace

func TestGraceCapsTheWholeSequence(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		gate, stuck := make(chan struct{}), make(chan struct{})
		defer close(gate)
		defer close(stuck)
		svc := &graceful.Service{
			Drain: 5 * time.Second, Requests: 10 * time.Second, Workers: 10 * time.Second, Grace: 12 * time.Second,
			Work: func(context.Context) { <-stuck }, // 不理 ctx 的背景工作
		}
		client, cancel, done := launch(t, svc, gate)

		go get(client, "/slow")
		synctest.Wait()
		begin := time.Now()
		cancel()

		if err := <-done; err == nil {
			t.Error("Run() = nil, want an error for the stuck request and worker")
		}
		if got := time.Since(begin); got != 12*time.Second {
			t.Errorf("shutdown took %v, want exactly Grace (12s); the step limits alone add up to 25s", got)
		}
	})
}

// ANCHOR_END: grace
