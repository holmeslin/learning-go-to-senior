// 示範 GOGC 不知道容器的記憶體上限：在 --memory 有上限的容器裡，這個程式會被 OOM 殺掉。
// 它刻意吃掉大量記憶體，只用 go run 手動執行，不放進測試。
package main

import (
	"fmt"
	"runtime/debug"
)

// ANCHOR: main

func main() {
	fmt.Println("memory limit:", debug.SetMemoryLimit(-1))
	cached := run(160, 2000)
	fmt.Println("done, cached MiB:", cached)
}

// ANCHOR_END: main
