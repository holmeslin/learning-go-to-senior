// 命令 after 是 multi-stage 映像打包的服務：收到 SIGTERM 時先停止接收新連線，再結束。
package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter12/04_container_image/after/service"
)

// ANCHOR: main

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer stop()

	srv := &http.Server{Addr: *addr, Handler: service.NewHandler(), ReadHeaderTimeout: 5 * time.Second}
	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe() }()
	slog.Info("listening", "addr", *addr, "uid", os.Getuid())

	select {
	case err := <-errc:
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	case <-ctx.Done():
	}

	slog.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("shutdown", "err", err)
		os.Exit(1)
	}
}

// ANCHOR_END: main
