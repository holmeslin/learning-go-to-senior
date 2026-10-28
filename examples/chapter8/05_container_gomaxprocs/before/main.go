package main

import (
	"fmt"
	"runtime"
)

// ANCHOR: main

func main() {
	// 舊版 Go 在容器裡看到的是整台機器的 CPU，所以照當時容器的 CPU limit（4 顆）寫死。
	runtime.GOMAXPROCS(4)

	fmt.Printf("NumCPU=%d GOMAXPROCS=%d\n", runtime.NumCPU(), runtime.GOMAXPROCS(0))
}

// ANCHOR_END: main
