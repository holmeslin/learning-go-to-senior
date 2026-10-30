// Package health 提供服務的健康檢查端點：/livez 回答行程還活著嗎，/readyz 回答能不能接流量。
package health

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Check 是 /readyz 要檢查的一項下游。
type Check struct {
	Name    string                          // 出現在回應裡的名稱
	Timeout time.Duration                   // 這一項檢查最多等多久
	Probe   func(ctx context.Context) error // 檢查本身；回傳 nil 代表下游可用
}

// NewHandler 回傳 /livez 與 /readyz 兩個端點。/livez 不檢查任何下游；
// /readyz 同時執行 checks，全部通過回 200，否則回 503 並列出失敗的檢查。
func NewHandler(checks ...Check) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /livez", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ok")
	})
	mux.Handle("GET /readyz", readiness(checks))
	return mux
}

// ANCHOR: ready

// readiness 把一組檢查變成 /readyz 的 handler。
type readiness []Check

func (checks readiness) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	errs := make([]error, len(checks))
	var wg sync.WaitGroup
	for i, c := range checks {
		wg.Go(func() { errs[i] = run(r.Context(), c) })
	}
	wg.Wait()

	report := struct {
		Status string            `json:"status"`
		Failed map[string]string `json:"failed,omitempty"`
	}{Status: "ok"}
	for i, err := range errs {
		if err != nil {
			if report.Failed == nil {
				report.Failed = map[string]string{}
			}
			report.Failed[checks[i].Name] = err.Error()
		}
	}
	code := http.StatusOK
	if report.Failed != nil {
		report.Status, code = "unavailable", http.StatusServiceUnavailable
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(report)
}

// run 執行一項檢查。就算 Probe 不理 ctx，超過 Timeout 也照樣回傳。
func run(ctx context.Context, c Check) error {
	ctx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()

	res := make(chan error, 1) // 有緩衝：逾時之後 Probe 才回來，也不會卡在送結果
	go func() { res <- c.Probe(ctx) }()

	var err error
	select {
	case err = <-res:
	case <-ctx.Done():
		err = ctx.Err()
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return errors.New("timeout")
	}
	return err
}

// ANCHOR_END: ready
