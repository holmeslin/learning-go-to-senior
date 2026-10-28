// 指令 accesscount 依規則統計一批存取紀錄，規則由第一個參數指定，例如 5xx 或 path:/api/。
package main

import (
	"fmt"
	"os"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter7/07_pgo/after/rules"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: accesscount <5xx|path:PREFIX>")
		os.Exit(2)
	}
	rule, err := rules.ParseRule(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	fmt.Printf("%+v\n", rules.Tally(rules.Sample(1000), rule))
}
