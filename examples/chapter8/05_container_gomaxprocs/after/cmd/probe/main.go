package main

import (
	"fmt"
	"os"
	"runtime"
	"runtime/debug"

	cpucheck "github.com/holmeslin/learning-go-to-senior/examples/chapter8/05_container_gomaxprocs/after"
)

// ANCHOR: main

func main() {
	fmt.Printf("NumCPU=%d GOMAXPROCS=%d\n", runtime.NumCPU(), runtime.GOMAXPROCS(0))
	if info, ok := debug.ReadBuildInfo(); ok {
		if err := cpucheck.Check(info, os.Getenv); err != nil {
			fmt.Println("warning:", err)
		}
	}
}

// ANCHOR_END: main
