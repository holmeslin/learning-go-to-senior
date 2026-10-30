// 命令 before 是單階段映像打包的服務：功能與 after 相同，但沒有處理 SIGTERM。
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	flag.Parse()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /livez", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ok")
	})
	mux.HandleFunc("GET /hello", func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		if name == "" {
			name = "world"
		}
		fmt.Fprintf(w, "hello, %s\n", name)
	})

	log.Printf("listening addr=%s uid=%d", *addr, os.Getuid())
	log.Fatal(http.ListenAndServe(*addr, mux))
}
