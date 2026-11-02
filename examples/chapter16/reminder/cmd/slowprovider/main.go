// Command slowprovider 是壓測用的慢速假推播供應商：可以設定每個請求的延遲與回 429 的比例，只聽 loopback。
//
//	go run ./examples/chapter16/reminder/cmd/slowprovider -addr 127.0.0.1:9000 -delay 200ms -rate429 0.1
//
// 把提醒服務的 REMINDER_PROVIDER_URL 指向它（例如 http://127.0.0.1:9000）。
// 它只在重現腳本裡執行，不屬於服務本體；旗標與 signal 只在這個 main 處理，行為在 internal/slowprovider。
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter16/reminder/internal/slowprovider"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	if err := run(log); err != nil {
		log.Error("slow provider stopped", "error", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	addr := flag.String("addr", "127.0.0.1:9000", "監聽位址，主機必須是 loopback")
	delay := flag.Duration("delay", 100*time.Millisecond, "每個請求回應前等待的時間")
	rate429 := flag.Float64("rate429", 0, "回 429 的比例，0 到 1")
	flag.Parse()

	if err := checkLoopback(*addr); err != nil {
		return fmt.Errorf("-addr: %w", err)
	}
	h, err := slowprovider.NewHandler(slowprovider.Options{Delay: *delay, TooManyRequests: *rate429})
	if err != nil {
		return err
	}
	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	srv := &http.Server{Handler: h, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		<-ctx.Done()
		// 假供應商沒有要保護的進行中請求，直接關閉；Shutdown 會等慢的請求，正好和這個工具的用途相反。
		_ = srv.Close()
	}()
	log.Info("slow provider listening", "addr", ln.Addr().String(), "delay", delay.String(), "rate429", *rate429)
	if err := srv.Serve(ln); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// checkLoopback 確認 addr 的主機是 loopback：假供應商不該被網路上的其他機器連到。
func checkLoopback(addr string) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}
	if host != "localhost" && !net.ParseIP(host).IsLoopback() {
		return errors.New("host must be a loopback address such as 127.0.0.1")
	}
	return nil
}
