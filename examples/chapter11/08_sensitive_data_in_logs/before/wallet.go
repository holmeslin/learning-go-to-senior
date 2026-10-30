// Package wallet 讓使用者綁定信用卡。
//
// 這是 before，是刻意示範的漏洞，不要照用：handler 把整個請求 struct 與整組標頭
// 交給 slog，密碼、token 與完整的卡號都會進日誌。
package wallet

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// BindRequest 是 POST /cards 的請求本文。
type BindRequest struct {
	// UserID 是使用者代號。
	UserID string `json:"user_id"`
	// Password 是再次確認身分用的密碼。
	Password string `json:"password"`
	// CardNumber 是要綁定的完整卡號。
	CardNumber string `json:"card_number"`
}

// ANCHOR: handler

// NewHandler 回傳處理 POST /cards 的 handler。為了「出事時能還原現場」，
// 它把整個請求與整組標頭都記進日誌。
func NewHandler(logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req BindRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		logger.Info("bind card", slog.Any("request", req), slog.Any("headers", r.Header))
		w.WriteHeader(http.StatusNoContent)
	})
}

// ANCHOR_END: handler
