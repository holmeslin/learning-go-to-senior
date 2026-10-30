// Command demo 在 :8080 啟動服務，收到 SIGINT 或 SIGTERM 時依序關機。
//
// 它會監聽 port 並處理 signal，只放在 main 套件：CI 只編譯，不執行。
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	graceful "github.com/holmeslin/learning-go-to-senior/examples/chapter10/06_graceful_shutdown/after"
)

// ANCHOR: main

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	context.AfterFunc(ctx, stop) // 收到訊號後恢復預設行為：關機卡住時，再按一次 Ctrl+C 就直接結束

	svc := &graceful.Service{
		Drain:    5 * time.Second,  // 探針的間隔乘上連續失敗的次數，再加一點餘裕
		Requests: 10 * time.Second, // 最慢的請求做得完的時間
		Workers:  5 * time.Second,  // 背景工作收尾所需
		Grace:    25 * time.Second, // 比平台的寬限期（例如 30 秒）短，留時間給行程退出
		Work: func(ctx context.Context) {
			<-ctx.Done() // 真正的背景工作在這裡處理佇列，ctx 結束時做完手上的一筆再回傳
		},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /readyz", svc.Ready)
	mux.HandleFunc("GET /slow", func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(3 * time.Second):
			fmt.Fprintln(w, "saved")
		case <-r.Context().Done():
		}
	})
	svc.Server = &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}

	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal(err)
	}
	if err := svc.Run(ctx, ln); err != nil {
		log.Fatal(err)
	}
}

// ANCHOR_END: main
