// 和 before 同一個負載，但啟動時由 cgroup 的 memory.max 設定 GOMEMLIMIT，在 --memory 有上限的容器裡撐得住。
// 它刻意吃掉大量記憶體，只用 go run 手動執行，不放進測試。
package main

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter8/03_gomemlimit_container/after/memlimit"
)

// ANCHOR: main

func main() {
	if _, err := memlimit.Apply(os.DirFS("/sys/fs/cgroup")); err != nil {
		fmt.Println("GOMEMLIMIT not applied:", err) // 沒有 cgroup、沒有上限或已經設過：維持原狀
	}
	fmt.Println("memory limit:", debug.SetMemoryLimit(-1))
	cached := run(160, 2000)
	fmt.Println("done, cached MiB:", cached)
}

// ANCHOR_END: main
