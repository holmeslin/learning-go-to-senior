// Package runtimestats 用 runtime/metrics 讀出服務要回報的 runtime 指標。
package runtimestats

import "runtime/metrics"

// ANCHOR: names

// Names 回傳服務要回報的指標，名稱與說明都來自 metrics.All。
// 每次回傳新的切片，呼叫端改動它不影響別人；加一個指標只要加一行名稱。
func Names() []string {
	return []string{
		"/memory/classes/total:bytes",        // runtime 向作業系統要的全部記憶體
		"/memory/classes/heap/objects:bytes", // 活著與還沒被回收的物件
		"/gc/heap/live:bytes",                // 上一次 GC 標記到的活物件
		"/gc/heap/goal:bytes",                // 這一輪 GC 的 heap 目標
		"/gc/cycles/total:gc-cycles",         // 完成的 GC 次數
		"/sched/goroutines:goroutines",       // 活著的 goroutine 數
		"/cpu/classes/gc/total:cpu-seconds",  // GC 用掉的 CPU 時間
		"/cpu/classes/total:cpu-seconds",     // 可用的 CPU 時間總和
	}
}

// ANCHOR_END: names

// ANCHOR: sampler

// Sampler 重複使用同一個樣本切片讀取指標；不能同時被多個 goroutine 呼叫。
type Sampler struct {
	samples     []metrics.Sample
	unsupported []string
}

// New 依名稱建立 Sampler。這個 Go 版本沒有的名稱、值不是單一數值的指標
// （直方圖）會記進 Unsupported，Read 時略過。
func New(names ...string) *Sampler {
	kinds := map[string]metrics.ValueKind{}
	for _, d := range metrics.All() {
		kinds[d.Name] = d.Kind
	}
	s := &Sampler{}
	for _, name := range names {
		switch kinds[name] {
		case metrics.KindUint64, metrics.KindFloat64:
			s.samples = append(s.samples, metrics.Sample{Name: name})
		default:
			s.unsupported = append(s.unsupported, name)
		}
	}
	return s
}

// Unsupported 回傳 New 略過的名稱，順序與傳入時相同。
func (s *Sampler) Unsupported() []string { return s.unsupported }

// Read 讀出所有支援的指標，依 Kind 轉成 float64 寫進 dst。
// 呼叫端重複使用同一個 dst，讀取就不會產生新的配置。
func (s *Sampler) Read(dst map[string]float64) {
	metrics.Read(s.samples)
	for _, sm := range s.samples {
		switch sm.Value.Kind() {
		case metrics.KindUint64:
			dst[sm.Name] = float64(sm.Value.Uint64())
		case metrics.KindFloat64:
			dst[sm.Name] = sm.Value.Float64()
		}
	}
}

// ANCHOR_END: sampler
