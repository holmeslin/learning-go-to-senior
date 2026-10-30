// Package graceful 讓服務依固定的順序關機：先讓 readiness 失敗，再等進行中的請求，最後停背景工作。
package graceful

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync/atomic"
	"time"
)

// Service 是一個要優雅關機的服務：一個 HTTP 伺服器加上一個背景工作。
// 零值的各個時間上限代表「不等」，使用前請依部署平台的寬限期設定。
type Service struct {
	Server *http.Server              // 由呼叫端建立，Handler 要包含 Ready
	Work   func(ctx context.Context) // 背景工作；ctx 結束代表該停了，回傳代表停好了

	Drain    time.Duration // readiness 失敗之後，等負載平衡器把實例移出流量
	Requests time.Duration // 等進行中的請求做完的上限
	Workers  time.Duration // 等背景工作停下來的上限
	Grace    time.Duration // 整段關機的上限，從收到結束訊號起算

	draining atomic.Bool
}

// Ready 是 readiness 的 handler：關機開始之後回 503，平常回 200。
func (s *Service) Ready(w http.ResponseWriter, r *http.Request) {
	if s.draining.Load() {
		http.Error(w, "shutting down", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// ANCHOR: run

// Run 在 ln 上提供服務並啟動背景工作，直到 ctx 結束，再依序關機：
// readiness 失敗並等 Drain，Shutdown 等進行中的請求，通知背景工作停止並等它。
// 每一步都有自己的上限，而且全部算在 Grace 之內；超過上限的步驟會強制結束，並反映在回傳的錯誤裡。
// ln 由呼叫端傳入，正式環境是真的 listener，測試可以換成記憶體裡的。
func (s *Service) Run(ctx context.Context, ln net.Listener) error {
	// 背景工作要比請求晚停，所以它的 ctx 不跟著 ctx 結束，但保留其中的值。
	workCtx, stopWork := context.WithCancel(context.WithoutCancel(ctx))
	defer stopWork()
	workDone := make(chan struct{})
	go func() { defer close(workDone); s.Work(workCtx) }()

	stopWorkers := func(parent context.Context) error {
		stopWork()
		wctx, cancel := context.WithTimeout(parent, s.Workers)
		defer cancel()
		select {
		case <-workDone:
			return nil
		case <-wctx.Done():
			return fmt.Errorf("stop workers: %w", wctx.Err())
		}
	}

	serveErr := make(chan error, 1)
	go func() { serveErr <- s.Server.Serve(ln) }()
	select {
	case err := <-serveErr: // 伺服器自己失敗，不用走關機的前兩步
		return errors.Join(err, stopWorkers(context.Background()))
	case <-ctx.Done():
	}

	// 寬限期從這一刻起算；不能用已經結束的 ctx，否則每一步都會立刻放棄。
	grace, cancel := context.WithTimeout(context.Background(), s.Grace)
	defer cancel()

	s.draining.Store(true)
	drain, cancelDrain := context.WithTimeout(grace, s.Drain)
	<-drain.Done()
	cancelDrain()

	reqs, cancelReqs := context.WithTimeout(grace, s.Requests)
	defer cancelReqs()
	var reqErr error
	if err := s.Server.Shutdown(reqs); err != nil {
		_ = s.Server.Close() // 超過上限：切斷還沒做完的連線
		reqErr = fmt.Errorf("shutdown requests: %w", err)
	}
	return errors.Join(reqErr, stopWorkers(grace))
}

// ANCHOR_END: run
