// 指令 ingest 用固定的負載反覆呼叫 Handle，配合 GODEBUG=gctrace=1 觀察 GC；
// 它會長期占用記憶體與 CPU，只給手動執行，不放進測試。
package main

import (
	"flag"
	"fmt"
	"os"

	after "github.com/holmeslin/learning-go-to-senior/examples/chapter8/02_when_to_tune_gogc/after"
	before "github.com/holmeslin/learning-go-to-senior/examples/chapter8/02_when_to_tune_gogc/before"
)

func main() {
	impl := flag.String("impl", "after", "before（每批手動 GC）或 after")
	members := flag.Int("members", 200_000, "長駐在記憶體的會員數")
	batches := flag.Int("batches", 3000, "處理幾批")
	size := flag.Int("size", 2000, "每批的 ID 數")
	flag.Parse()

	var handle func([]int) []byte
	switch *impl {
	case "before":
		handle = before.Load(*members).Handle
	case "after":
		handle = after.Load(*members).Handle
	default:
		fmt.Fprintf(os.Stderr, "unknown impl %q\n", *impl)
		os.Exit(2)
	}

	ids := make([]int, *size)
	total := 0
	for b := range *batches {
		for i := range ids {
			ids[i] = (b**size + i) % *members
		}
		total += len(handle(ids))
	}
	fmt.Printf("impl=%s batches=%d report_bytes=%d\n", *impl, *batches, total)
}
