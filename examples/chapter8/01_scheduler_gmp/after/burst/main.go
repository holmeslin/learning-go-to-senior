// 指令 burst 對大量 blob 呼叫 blobsum.Sum，給 GODEBUG=schedtrace 觀察排程器用。
package main

import (
	"fmt"

	blobsum "github.com/holmeslin/learning-go-to-senior/examples/chapter8/01_scheduler_gmp/after"
)

func main() {
	buf := make([]byte, 64<<10)
	blobs := make([][]byte, 200_000)
	for i := range blobs {
		blobs[i] = buf
	}
	fmt.Println(len(blobsum.Sum(blobs)))
}
