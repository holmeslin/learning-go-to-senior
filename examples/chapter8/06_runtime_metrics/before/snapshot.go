// Package runtimestats 定期呼叫 runtime.ReadMemStats，自己湊出要回報的記憶體指標。
package runtimestats

import "runtime"

// ANCHOR: snapshot

// Snapshot 回傳這一刻的記憶體指標，key 是自訂的名稱。
func Snapshot() map[string]float64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return map[string]float64{
		"heap_bytes":       float64(m.Sys),       // heap 用了多少記憶體
		"heap_alloc_bytes": float64(m.HeapAlloc), // 已配置的物件
		"gc_runs":          float64(m.NumGC),
		"goroutines":       float64(runtime.NumGoroutine()),
	}
}

// ANCHOR_END: snapshot
