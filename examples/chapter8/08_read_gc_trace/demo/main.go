// Command demo 模擬一個有常駐快取的服務，產生給 GODEBUG=gctrace=1 讀的 GC trace。
//
// 這是會吃大量記憶體的示範，只放在 main 套件：CI 只編譯，不執行。
package main

import (
	"flag"
	"runtime"
	"sync"
	"sync/atomic"
)

// ANCHOR: demo

// entry 是 128 位元組、含指標的物件：GC 標記時必須逐個掃描它。
type entry struct {
	next    *entry
	payload [120]byte
}

// last 讓編譯器無法把用完就丟的緩衝區優化掉，它們一定會配置在 heap 上。
var last atomic.Pointer[[]byte]

func main() {
	live := flag.Int("live", 64, "常駐快取的大小（MiB），整段執行都活著")
	total := flag.Int("total", 2048, "用完就丟的回應緩衝區，累計配置多少 MiB")
	flag.Parse()

	// 常駐的資料：GC 每一輪都要標記它，所以活著的量決定一輪的成本與 heap 目標。
	cache := make([]*entry, *live*8192)
	for i := range cache {
		cache[i] = &entry{next: cache[max(i-1, 0)]}
	}

	// 每個 worker 不斷配置 32 KiB 的緩衝區，用完就成為垃圾。
	const size = 32 << 10
	perWorker := *total << 20 / runtime.GOMAXPROCS(0)
	var wg sync.WaitGroup
	for range runtime.GOMAXPROCS(0) {
		wg.Go(func() {
			for n := 0; n < perWorker; n += size {
				buf := make([]byte, size)
				last.Store(&buf)
			}
		})
	}
	wg.Wait()
	runtime.KeepAlive(cache)
}

// ANCHOR_END: demo
