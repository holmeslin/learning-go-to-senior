// Command versiond 是示範版本資訊的小服務（常見寫法）。
// 版本字串靠建置指令的 -X 注入，和實際建置的 commit 沒有任何關係，不要照用。
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"
)

// ANCHOR: version

// version 由建置指令的 -ldflags "-X main.version=..." 注入；沒有注入時是 "dev"。
var version = "dev"

// ANCHOR_END: version

func main() {
	showVersion := flag.Bool("version", false, "印出版本後結束")
	addr := flag.String("addr", "127.0.0.1:8080", "監聽位址")
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		return
	}

	slog.Info("starting", "addr", *addr, "version", version)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /version", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintln(w, version)
	})
	srv := &http.Server{Addr: *addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	if err := srv.ListenAndServe(); err != nil {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}
