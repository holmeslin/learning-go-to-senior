// Package graceful 是常見寫法：收到結束訊號就直接結束，或只等 HTTP 伺服器。
package graceful

import (
	"context"
	"net"
	"net/http"
	"time"
)

// ANCHOR: exit

// Exit 啟動背景工作與伺服器，ctx 結束（收到訊號）就回傳；呼叫它的 main 接著結束行程。
func Exit(ctx context.Context, srv *http.Server, ln net.Listener, work func(ctx context.Context)) error {
	go work(ctx)
	go srv.Serve(ln)
	<-ctx.Done()
	return nil
}

// ANCHOR_END: exit

// ANCHOR: shutdown

// ShutdownOnly 在 ctx 結束時呼叫 Shutdown 等進行中的請求，但沒有等背景工作。
func ShutdownOnly(ctx context.Context, srv *http.Server, ln net.Listener, work func(ctx context.Context)) error {
	go work(ctx)
	go srv.Serve(ln)
	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

// ANCHOR_END: shutdown
