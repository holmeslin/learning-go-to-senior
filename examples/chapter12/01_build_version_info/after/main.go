// Command versiond 是示範版本資訊的小服務：-version 印出版本，/version 端點與啟動日誌也有。
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"runtime/debug"
	"time"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter12/01_build_version_info/after/version"
)

// ANCHOR: build

// build 只放建置系統才知道的值，連結時用 -ldflags "-X main.build=..." 注入；
// 沒有注入就是空字串。commit、時間與版本不放這裡，go build 已經記在執行檔裡。
var build string

// ANCHOR_END: build

// ANCHOR: main

func main() {
	showVersion := flag.Bool("version", false, "印出版本資訊後結束")
	addr := flag.String("addr", "127.0.0.1:8080", "監聽位址")
	flag.Parse()

	bi, _ := debug.ReadBuildInfo() // 讀不到時是 nil，FromBuildInfo 會處理
	info := version.FromBuildInfo(bi, build)
	if *showVersion {
		fmt.Println(info)
		return
	}

	slog.Info("starting", "addr", *addr, "version", info)
	mux := http.NewServeMux()
	mux.Handle("GET /version", version.Handler(info))
	srv := &http.Server{Addr: *addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	if err := srv.ListenAndServe(); err != nil {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

// ANCHOR_END: main
