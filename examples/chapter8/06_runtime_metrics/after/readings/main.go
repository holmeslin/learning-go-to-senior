// readings 並排印出同一個時間點的兩種讀法：ReadMemStats 湊出來的指標，與 runtime/metrics 的指標。
// 它會配置 64 MiB 的資料，只給讀者手動執行，CI 只編譯。
package main

import (
	"fmt"
	"runtime"

	after "github.com/holmeslin/learning-go-to-senior/examples/chapter8/06_runtime_metrics/after"
	before "github.com/holmeslin/learning-go-to-senior/examples/chapter8/06_runtime_metrics/before"
)

const mib = 1 << 20

// rows 是要並排印出的指標；來源是 "before" 或 "after"。
var rows = []struct{ source, name string }{
	{"before", "heap_bytes"},
	{"before", "heap_alloc_bytes"},
	{"after", "/memory/classes/total:bytes"},
	{"after", "/memory/classes/heap/objects:bytes"},
	{"after", "/gc/heap/live:bytes"},
}

func main() {
	s := after.New(after.Names...)
	dst := map[string]float64{}
	var cols []map[string]float64

	read := func() {
		col := map[string]float64{}
		for k, v := range before.Snapshot() {
			col["before/"+k] = v
		}
		s.Read(dst)
		for k, v := range dst {
			col["after/"+k] = v
		}
		cols = append(cols, col)
	}

	runtime.GC()
	read() // 剛啟動

	data := make([][]byte, 64)
	for i := range data {
		data[i] = make([]byte, mib)
		data[i][0] = 1
	}
	runtime.GC() // 讓 /gc/heap/live 反映這一刻
	read()       // 持有 64 MiB

	runtime.KeepAlive(data)
	data = nil
	runtime.GC()
	read() // 放掉並 GC

	fmt.Printf("P=%d\n%-7s %-38s %7s %7s %8s\n", runtime.GOMAXPROCS(0), "source", "metric (MiB)", "start", "hold", "released")
	for _, r := range rows {
		key := r.source + "/" + r.name
		fmt.Printf("%-7s %-38s %7.1f %7.1f %8.1f\n", r.source, r.name, cols[0][key]/mib, cols[1][key]/mib, cols[2][key]/mib)
	}
}
