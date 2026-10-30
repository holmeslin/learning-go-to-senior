// Command slugtool 印出自己是由哪個模組、哪個版本建出來的，
// 用來驗證 go install 裝到的是不是預期的版本。
package main

import (
	"fmt"
	"runtime/debug"
)

func main() {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		fmt.Println("no build info")
		return
	}
	fmt.Println(info.Main.Path, info.Main.Version)
}
