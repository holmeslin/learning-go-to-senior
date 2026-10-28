// 命令 demo 對照 before 與 after 的 heap 用量，會刻意配置數百 MiB，只給手動 go run，CI 只編譯。
//
// 用法：
//
//	go run ./examples/chapter8/04_heap_growth/after/demo -case=cache|seen|pending -mode=before|after
//	go run ./examples/chapter8/04_heap_growth/after/demo -case=rss
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"runtime/metrics"
	"strconv"
	"strings"

	after "github.com/holmeslin/learning-go-to-senior/examples/chapter8/04_heap_growth/after"
	before "github.com/holmeslin/learning-go-to-senior/examples/chapter8/04_heap_growth/before"
)

const mib = 1 << 20

func main() {
	which := flag.String("case", "cache", "cache、seen、pending 或 rss")
	mode := flag.String("mode", "before", "before 或 after")
	flag.Parse()

	fmt.Println("GOMAXPROCS", runtime.GOMAXPROCS(0))
	isAfter := *mode == "after"
	switch *which {
	case "cache":
		cacheCase(isAfter)
	case "seen":
		seenCase(isAfter)
	case "pending":
		pendingCase(isAfter)
	case "rss":
		rssCase()
	default:
		fmt.Fprintf(os.Stderr, "unknown case %q\n", *which)
		os.Exit(2)
	}
}

// 50 萬個客戶各算一次摘要。
func cacheCase(isAfter bool) {
	var put func(key, value string)
	var keep any
	if isAfter {
		c := after.NewCache(10_000)
		put, keep = c.Put, c
	} else {
		c := before.NewCache()
		put, keep = c.Put, c
	}
	for i := range 500_000 {
		key := strconv.Itoa(i)
		put(key, key+strings.Repeat("x", 64))
	}
	report(label(isAfter, "cache"))
	runtime.KeepAlive(keep)
}

// 2000 筆 64 KiB 的通知，各自只留 16 位元組的請求代號。
func seenCase(isAfter bool) {
	var add func([]byte) error
	var keep any
	if isAfter {
		s := new(after.Seen)
		add, keep = s.Add, s
	} else {
		s := new(before.Seen)
		add, keep = s.Add, s
	}
	for range 2000 {
		if err := add(make([]byte, 64<<10)); err != nil {
			panic(err)
		}
	}
	report(label(isAfter, "seen"))
	runtime.KeepAlive(keep)
}

// 尖峰時 100 萬筆等待中，之後只剩 1000 筆。
func pendingCase(isAfter bool) {
	const peak, left, endpoint = 1_000_000, 1000, "https://example.com/hook"
	var start, done func(uint64)
	compact := func() {}
	var keep any
	if isAfter {
		p := after.NewPending()
		start = func(id uint64) { p.Start(id, after.Delivery{Endpoint: endpoint}) }
		done, compact, keep = p.Done, p.Compact, p
	} else {
		p := before.NewPending()
		start = func(id uint64) { p.Start(id, before.Delivery{Endpoint: endpoint}) }
		done, keep = p.Done, p
	}
	for id := range uint64(peak) {
		start(id)
	}
	for id := range uint64(peak - left) {
		done(id)
	}
	compact()
	report(label(isAfter, "pending"))
	runtime.KeepAlive(keep)
}

// 配置 256 MiB 再全部放掉，看 heap 與行程的 RSS 各自什麼時候下降。
func rssCase() {
	chunks := make([][]byte, 256)
	for i := range chunks {
		chunks[i] = make([]byte, mib)
		clear(chunks[i]) // 寫過才算真的用了實體記憶體
	}
	runtime.GC() // /gc/heap/live:bytes 是上一次 GC 標記的結果，先 GC 才讀得到剛配置的
	snapshot("holding 256 MiB")
	runtime.KeepAlive(chunks)

	chunks = nil
	runtime.GC()
	snapshot("after runtime.GC()")

	debug.FreeOSMemory()
	snapshot("after debug.FreeOSMemory()")
}

func label(isAfter bool, name string) string {
	if isAfter {
		return "after  " + name
	}
	return "before " + name
}

// report 先 GC 一次，讓 /gc/heap/live:bytes 反映「還活著」的資料；這只是為了量測，不是調校手法。
func report(name string) {
	runtime.GC()
	s := []metrics.Sample{{Name: "/gc/heap/live:bytes"}}
	metrics.Read(s)
	fmt.Printf("%-15s live heap %7.1f MiB\n", name, float64(s[0].Value.Uint64())/mib)
}

func snapshot(step string) {
	s := []metrics.Sample{
		{Name: "/gc/heap/live:bytes"},
		{Name: "/memory/classes/heap/free:bytes"},
		{Name: "/memory/classes/heap/released:bytes"},
	}
	metrics.Read(s)
	fmt.Printf("%-27s live %6.1f  free %6.1f  released %6.1f  VmRSS %s\n", step,
		float64(s[0].Value.Uint64())/mib, float64(s[1].Value.Uint64())/mib,
		float64(s[2].Value.Uint64())/mib, vmRSS())
}

// vmRSS 讀 /proc/self/status 的 VmRSS；沒有這個檔案（不是 Linux）就回傳 n/a。
func vmRSS() string {
	f, err := os.Open("/proc/self/status")
	if err != nil {
		return "n/a"
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if v, ok := strings.CutPrefix(sc.Text(), "VmRSS:"); ok {
			kb, err := strconv.Atoi(strings.TrimSuffix(strings.TrimSpace(v), " kB"))
			if err != nil {
				return "n/a"
			}
			return fmt.Sprintf("%.1f MiB", float64(kb)/1024)
		}
	}
	return "n/a"
}
