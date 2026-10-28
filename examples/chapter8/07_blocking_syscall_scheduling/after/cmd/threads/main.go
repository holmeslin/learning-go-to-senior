//go:build unix

// Command threads 示範阻塞式 syscall 與網路 I/O 對作業系統執行緒數的影響。
//
// 它會同時卡住 n 條讀取，只能手動執行，不要放進測試。
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"syscall"

	after "github.com/holmeslin/learning-go-to-senior/examples/chapter8/07_blocking_syscall_scheduling/after"
	before "github.com/holmeslin/learning-go-to-senior/examples/chapter8/07_blocking_syscall_scheduling/before"
)

// ANCHOR: main

func main() {
	mode := flag.String("mode", "pipe", "pipe：syscall 讀管線；net：讀 TCP 連線")
	n := flag.Int("n", 64, "同時要讀的來源數")
	limit := flag.Int("limit", 0, "同時讀取的上限；0 表示不限（before）")
	maxThreads := flag.Int("maxthreads", 0, "大於 0 時呼叫 debug.SetMaxThreads")
	flag.Parse()
	if *maxThreads > 0 {
		debug.SetMaxThreads(*maxThreads)
	}

	entered := make(chan struct{}, *n)
	var read func(context.Context, int) (int, error)
	var release func(int)
	switch *mode {
	case "pipe":
		// syscall.Pipe 給的是阻塞式的檔案描述子：沒有資料時 syscall.Read 會占住執行緒。
		rfd, wfd := make([]int, *n), make([]int, *n)
		for i := range *n {
			var p [2]int
			if err := syscall.Pipe(p[:]); err != nil {
				log.Fatal(err)
			}
			rfd[i], wfd[i] = p[0], p[1]
		}
		read = func(ctx context.Context, i int) (int, error) {
			entered <- struct{}{}
			return syscall.Read(rfd[i], make([]byte, 1))
		}
		release = func(i int) { syscall.Write(wfd[i], []byte{1}) }
	case "net":
		// 網路連線由 netpoller 管：沒有資料時，goroutine 停放，執行緒不被占住。
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			log.Fatal(err)
		}
		clients, servers := make([]net.Conn, *n), make([]net.Conn, *n)
		for i := range *n {
			if clients[i], err = net.Dial("tcp", ln.Addr().String()); err != nil {
				log.Fatal(err)
			}
			if servers[i], err = ln.Accept(); err != nil {
				log.Fatal(err)
			}
		}
		read = func(ctx context.Context, i int) (int, error) {
			entered <- struct{}{}
			return clients[i].Read(make([]byte, 1))
		}
		release = func(i int) { servers[i].Write([]byte{1}) }
	default:
		log.Fatalf("unknown mode %q", *mode)
	}

	threads := func() int { return pprof.Lookup("threadcreate").Count() }
	start := threads()
	sources := make([]int, *n)
	for i := range sources {
		sources[i] = i
	}
	done := make(chan int)
	go func() {
		var total int
		var err error
		if *limit > 0 {
			total, err = after.ReadAll(context.Background(), sources, *limit, read)
		} else {
			total, err = before.ReadAll(context.Background(), sources, read)
		}
		if err != nil {
			log.Fatal(err)
		}
		done <- total
	}()

	want := *n // 有上限時，同時只會有 limit 條進得了 read
	if *limit > 0 {
		want = min(*n, *limit)
	}
	for range want {
		<-entered // 等每一條讀取都進了 read，再量執行緒數
	}
	fmt.Printf("mode=%s sources=%d limit=%d GOMAXPROCS=%d threads created: %d -> %d\n",
		*mode, *n, *limit, runtime.GOMAXPROCS(0), start, threads())
	for i := range sources {
		release(i)
	}
	fmt.Println("bytes read:", <-done)
}

// ANCHOR_END: main
