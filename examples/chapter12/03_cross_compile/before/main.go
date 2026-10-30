// 命令 probe 印出所在主機的基本資訊。這是刻意示範的寫法：主機名稱靠 cgo、
// 檔案數上限直接呼叫 unix 才有的 syscall，兩者都讓交叉編譯失敗。
package main

import "fmt"

func main() {
	limit, err := fileLimit()
	if err != nil {
		fmt.Println("probe:", err)
		return
	}
	fmt.Printf("host=%s nofile=%d\n", hostname(), limit)
}
