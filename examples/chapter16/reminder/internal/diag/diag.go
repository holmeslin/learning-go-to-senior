// Package diag 是提醒服務的診斷端點：健康檢查、readiness、pprof 與指標快照。
//
// 它掛在自己的 ServeMux 上，和內部 API 是兩個 handler、兩個 listener；
// 呼叫端只能把它交給只聽 loopback 的位址，因為 pprof 不能對外。
package diag

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/pprof"
	"sync/atomic"
)

// Readiness 是服務「能不能接新工作」的旗標。零值是未就緒：服務啟動完成才設成就緒，
// 關機一開始就設回未就緒，讓負載平衡先把流量移走，再停掃描與關伺服器。可以在多個 goroutine 之間共用。
type Readiness struct {
	ready atomic.Bool
}

// Set 設定是否就緒。
func (r *Readiness) Set(ready bool) { r.ready.Store(ready) }

// Ready 回傳目前是否就緒。
func (r *Readiness) Ready() bool { return r.ready.Load() }

// ANCHOR: snapshotter

// Snapshotter 把指標的快照寫進 w。介面定義在使用端：診斷端點不需要知道指標從哪裡收。
type Snapshotter interface {
	Snapshot(ctx context.Context, w io.Writer) error
}

// ANCHOR_END: snapshotter

// ANCHOR: handler

// NewHandler 回傳診斷端點的 handler：
//
//	GET /healthz              行程還活著就是 200
//	GET /readyz               就緒 200，未就緒（啟動中、關機中）503
//	GET /debug/metrics        指標快照，純文字
//	GET /debug/pprof/...      具名的 profile、profile、symbol、trace；沒有 cmdline
//
// 這個伺服器不能設 WriteTimeout：/debug/pprof/profile 預設要跑 30 秒才回應。
func NewHandler(ready *Readiness, metrics Snapshotter) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, "ok\n")
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, _ *http.Request) {
		if !ready.Ready() {
			http.Error(w, "not ready\n", http.StatusServiceUnavailable)
			return
		}
		io.WriteString(w, "ready\n")
	})
	mux.HandleFunc("GET /debug/metrics", func(w http.ResponseWriter, r *http.Request) {
		// 先寫進緩衝區：收集失敗時才來得及回 500，不是送出一半的快照。
		var buf bytes.Buffer
		if err := metrics.Snapshot(r.Context(), &buf); err != nil {
			http.Error(w, "snapshot failed\n", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write(buf.Bytes())
	})
	mux.HandleFunc("GET /debug/pprof/", pprof.Index) // 也負責 heap、goroutine 等具名的 profile
	mux.HandleFunc("GET /debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("GET /debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("GET /debug/pprof/trace", pprof.Trace)
	return mux
}

// ANCHOR_END: handler
