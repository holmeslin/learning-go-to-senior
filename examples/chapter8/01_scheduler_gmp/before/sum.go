// Package blobsum 對一批上傳的檔案內容算 SHA-256，供去重複用。
package blobsum

import (
	"crypto/sha256"
	"sync"
)

// ANCHOR: sum

// Sum 回傳每個 blob 的 SHA-256，順序與 blobs 相同。
// 每個 blob 開一個 goroutine，同時有多少個在跑沒有上限。
func Sum(blobs [][]byte) [][sha256.Size]byte {
	sums := make([][sha256.Size]byte, len(blobs))
	var wg sync.WaitGroup
	for i, b := range blobs {
		wg.Go(func() { sums[i] = sha256.Sum256(b) })
	}
	wg.Wait()
	return sums
}

// ANCHOR_END: sum
