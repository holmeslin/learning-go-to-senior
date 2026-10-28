package main

// ANCHOR: run

// run 模擬一個把資料集留在記憶體裡的服務：前 cacheMiB 個請求的結果進快取、一直留著，
// 其餘請求只配置一塊 1 MiB 的暫存緩衝區，用完就丟。
func run(cacheMiB, requests int) int {
	cache := make([][]byte, 0, cacheMiB)
	for i := range requests {
		buf := make([]byte, 1<<20)
		for j := range buf {
			buf[j] = byte(i) // 寫入每個位元組，記憶體才會真的被占用
		}
		if i < cacheMiB {
			cache = append(cache, buf)
		}
	}
	return len(cache)
}

// ANCHOR_END: run
