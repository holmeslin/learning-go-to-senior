// 命令 probe 印出所在主機的基本資訊，用來確認交叉編譯出來的執行檔跑在哪個平台。
package main

import (
	"fmt"
	"os"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter12/03_cross_compile/after/probe"
)

func main() {
	info, err := probe.Collect()
	if err != nil {
		fmt.Fprintln(os.Stderr, "probe:", err)
		os.Exit(1)
	}
	fmt.Println(info)
}
