// Command slowpush 在本機重現提醒 worker 的事故：一個本機的慢速假供應商，加上每隔一陣就送出一批提醒的 worker。
// 用 -mode 選事故版本（before）或修正版本（after）；診斷端點掛在獨立的 ServeMux，只聽 loopback。
//
// 它會一直送出提醒，只在示範時執行；監聽 port 與處理 signal 的程式只放在 main 套件，CI 只編譯、不執行。
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	after "github.com/holmeslin/learning-go-to-senior/examples/chapter15/03_pprof_trace_live/after"
	before "github.com/holmeslin/learning-go-to-senior/examples/chapter15/03_pprof_trace_live/before"
)

func main() {
	mode := flag.String("mode", "before", "before 是事故版本，after 是修正版本")
	admin := flag.String("admin", "127.0.0.1:18150", "診斷端點的位址，只聽 loopback")
	supplier := flag.String("supplier", "127.0.0.1:18151", "慢速假供應商的位址，只聽 loopback")
	delay := flag.Duration("delay", time.Hour, "假供應商多久之後才回應；預設一小時，等於不回應")
	every := flag.Duration("every", 500*time.Millisecond, "worker 每隔多久送出一批提醒")
	batch := flag.Int("batch", 10, "每一批有幾則提醒")
	flag.Parse()

	send, err := newSend(*mode, "http://"+*supplier+"/push")
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serve(*supplier, slowSupplier(*delay))
	serve(*admin, adminHandler())
	log.Printf("mode=%s admin=%s supplier=%s delay=%v every=%v batch=%d", *mode, *admin, *supplier, *delay, *every, *batch)

	tick := time.NewTicker(*every)
	defer tick.Stop()
	for n := 1; ; n++ {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
		keys := make([]string, *batch)
		for i := range keys {
			keys[i] = fmt.Sprintf("event-%d/attendee-%d", n, i)
		}
		start := time.Now()
		err := send(ctx, keys)
		// before 的 SendAll 立刻回傳 nil，日誌看不出異常；after 要等到逾時才回傳。
		log.Printf("batch %d: took %v, failed=%t", n, time.Since(start).Round(time.Millisecond), err != nil)
	}
}

// newSend 依 mode 選出 before 或 after 的 Sender；兩個套件各有自己的 Push 型別，所以包成接受 key 的函式。
func newSend(mode, url string) (func(context.Context, []string) error, error) {
	client := &http.Client{} // 刻意不設 Timeout：逾時是 after 的 SendAll 自己負責的事
	switch mode {
	case "before":
		s := before.Sender{Client: client, URL: url}
		return func(ctx context.Context, keys []string) error {
			pushes := make([]before.Push, len(keys))
			for i, k := range keys {
				pushes[i] = before.Push{Key: k}
			}
			return s.SendAll(ctx, pushes)
		}, nil
	case "after":
		s := after.Sender{Client: client, URL: url}
		return func(ctx context.Context, keys []string) error {
			pushes := make([]after.Push, len(keys))
			for i, k := range keys {
				pushes[i] = after.Push{Key: k}
			}
			return s.SendAll(ctx, pushes)
		}, nil
	}
	return nil, errors.New("-mode must be before or after")
}

// slowSupplier 是慢速的假供應商：收到請求後等 delay 才回 204；呼叫端先放棄（連線關掉）就跟著結束。
func slowSupplier(delay time.Duration) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t := time.NewTimer(delay)
		defer t.Stop()
		select {
		case <-t.C:
			w.WriteHeader(http.StatusNoContent)
		case <-r.Context().Done():
		}
	})
}

// adminHandler 只註冊明確要開的診斷端點，做法同第 9 章〈/debug/pprof 與診斷端點〉。
func adminHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /debug/pprof/", pprof.Index) // 也負責 heap、goroutine 等具名的 profile
	mux.HandleFunc("GET /debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("GET /debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("GET /debug/pprof/trace", pprof.Trace)
	return mux
}

// serve 在 addr 上用 h 開一個 server；監聽失敗就結束程式。
// 不設 WriteTimeout：CPU profile 與 trace 要一直寫到 seconds 結束。
func serve(addr string, h http.Handler) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	srv := &http.Server{Handler: h, ReadHeaderTimeout: 5 * time.Second}
	go func() { log.Fatal(srv.Serve(ln)) }()
}
