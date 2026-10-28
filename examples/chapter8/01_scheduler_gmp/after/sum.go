// Package blobsum 對一批上傳的檔案內容算 SHA-256，供去重複用。
package blobsum

import (
	"crypto/sha256"
	"runtime"
	"sync"
	"sync/atomic"
)

// ANCHOR: foreach

// ForEach 對 0 到 n-1 的每個 i 呼叫 fn(i)，等全部做完才回傳。
// fn 應該是 CPU 密集的工作；同時執行的 fn 不超過呼叫當下的 runtime.GOMAXPROCS(0)。
func ForEach(n int, fn func(i int)) {
	var next atomic.Int64
	var wg sync.WaitGroup
	for range min(runtime.GOMAXPROCS(0), n) {
		wg.Go(func() {
			for {
				i := int(next.Add(1)) - 1
				if i >= n {
					return
				}
				fn(i)
			}
		})
	}
	wg.Wait()
}

// ANCHOR_END: foreach

// ANCHOR: sum

// Sum 回傳每個 blob 的 SHA-256，順序與 blobs 相同。
func Sum(blobs [][]byte) [][sha256.Size]byte {
	sums := make([][sha256.Size]byte, len(blobs))
	ForEach(len(blobs), func(i int) { sums[i] = sha256.Sum256(blobs[i]) })
	return sums
}

// ANCHOR_END: sum
