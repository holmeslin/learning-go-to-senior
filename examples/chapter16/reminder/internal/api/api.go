// Package api 是提醒服務的內部 HTTP API：唯讀，查詢某場活動各狀態的提醒數量，
// 給客服與產品負責人回答「這場活動的提醒送到哪了」。
//
// 契約：
//
//	GET /v1/events/{event_id}/reminder-counts
//	200 {"event_id":42,"counts":{"failed":5,"retrying":15,"sending":0,"sent":19980}}
//	400 event_id 不是正整數的標準寫法      404 沒有這場活動
//	503 查詢逾時（可以稍後重試）            500 其他錯誤，本文不帶內部的錯誤訊息
//	405 任何改變資料的方法；API 沒有寫入的操作
//
// 回應只有數量，沒有報名者、裝置 token 這類個人資料。
//
// shortcut: 沒有驗證與授權，只靠網路邊界（內部網路，不對外），範例的內部 API 預設監聽 :8080；
// 要開放給更多人或跨網路使用時，加上服務之間的身分驗證（第 11 章）。
package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter16/reminder/internal/store"
)

// ANCHOR: counter

// Counter 回傳某場活動各狀態的寄送紀錄數量。介面定義在使用端：API 只需要「數」這一個動作，
// *store.Store 滿足它，測試用一個直接回傳結果的假實作。
// 沒有這場活動時，回傳的錯誤要包著 store.ErrNotFound。
type Counter interface {
	CountByState(ctx context.Context, eventID int64) (map[store.State]int, error)
}

// ANCHOR_END: counter

// ANCHOR: handler

// NewHandler 回傳內部 API 的 handler。每次查詢的逾時是 queryTimeout：資料庫卡住時，請求不會跟著卡住；
// 500 的原因只寫進 log，不回給呼叫端。
func NewHandler(c Counter, queryTimeout time.Duration, log *slog.Logger) http.Handler {
	h := &handler{counter: c, queryTimeout: queryTimeout, log: log}
	mux := http.NewServeMux()
	// 只註冊 GET：ServeMux 對同一路徑的其他方法自動回 405 並帶 Allow，所以不需要自己擋寫入。
	mux.HandleFunc("GET /v1/events/{event_id}/reminder-counts", h.counts)
	return mux
}

type handler struct {
	counter      Counter
	queryTimeout time.Duration
	log          *slog.Logger
}

// response 是成功的回應。counts 的鍵由 encoding/json 排序，輸出是確定的。
type response struct {
	EventID int64               `json:"event_id"`
	Counts  map[store.State]int `json:"counts"`
}

func (h *handler) counts(w http.ResponseWriter, r *http.Request) {
	eventID, ok := parseEventID(r.PathValue("event_id"))
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "event_id must be a positive integer"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), h.queryTimeout)
	defer cancel()
	counts, err := h.counter.CountByState(ctx, eventID)
	switch {
	case err == nil:
		writeJSON(w, http.StatusOK, response{EventID: eventID, Counts: counts})
	case errors.Is(err, store.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "event not found"})
	case errors.Is(err, context.DeadlineExceeded):
		h.log.Warn("count reminders timed out", "event", eventID, "error", err)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "query timed out, try again later"})
	default:
		h.log.Error("count reminders failed", "event", eventID, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}
}

// parseEventID 只接受正整數的標準寫法：+5、007 雖然 ParseInt 讀得懂，
// 但同一場活動會有好幾個網址，快取與日誌都會被打散。
func parseEventID(s string) (int64, bool) {
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil || id <= 0 || strconv.FormatInt(id, 10) != s {
		return 0, false
	}
	return id, true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// ANCHOR_END: handler
