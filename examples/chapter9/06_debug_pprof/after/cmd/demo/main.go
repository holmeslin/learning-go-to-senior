// Command demo 啟動兩個 listener：對外的服務與只聽本機的管理端點，並製造三種症狀
// （一顆核心被吃滿、常駐的記憶體、卡住不退出的 goroutine）讓讀者收 profile。
//
// 它會一直吃滿一顆核心，只在示範時執行；監聽 port 的程式只放在 main 套件，CI 只編譯、不執行。
package main

import (
	"crypto/sha256"
	"flag"
	"log"
	"net/http"
	"runtime"
	"time"

	stock "github.com/holmeslin/learning-go-to-senior/examples/chapter9/06_debug_pprof/after"
)

// ANCHOR: main

func main() {
	addr := flag.String("addr", "127.0.0.1:18095", "對外服務的位址；正式環境對外的服務會聽 :8080 之類的位址")
	admin := flag.String("admin", "127.0.0.1:18096", "管理端點的位址，只聽本機或內部網路")
	flag.Parse()

	go burn()
	held := retain()
	leak(8)
	runtime.GC() // heap profile 以最近一次完成的 GC 為準，先收一次讓 retain 的記憶體算進去

	go func() {
		// 不設 WriteTimeout：CPU profile 與 trace 要一直寫到 seconds 結束。
		srv := &http.Server{Addr: *admin, Handler: stock.NewAdminHandler(), ReadHeaderTimeout: 5 * time.Second}
		log.Fatal(srv.ListenAndServe())
	}()
	srv := &http.Server{Addr: *addr, Handler: stock.NewPublicHandler(), ReadHeaderTimeout: 5 * time.Second}
	log.Printf("public %s, admin %s", *addr, *admin)
	err := srv.ListenAndServe()
	runtime.KeepAlive(held) // 讓編譯器不會把 retain 的配置當成沒人用而省掉
	log.Fatal(err)
}

// ANCHOR_END: main

// burn 不停地算雜湊，讓 CPU profile 看得到熱點。
func burn() {
	buf := make([]byte, 1<<10)
	for {
		sum := sha256.Sum256(buf)
		buf = append(buf[:0], sum[:]...)
	}
}

// retain 配置 32 MiB 並一直持有，讓 heap profile 的 inuse_space 看得到它。
func retain() []byte {
	return make([]byte, 32<<20)
}

// leak 開 n 個永遠等不到訊息的 goroutine，和第 6 章談過的洩漏同一個長相。
func leak(n int) {
	never := make(chan struct{})
	for range n {
		go func() { <-never }()
	}
}
