// Package wallet 讓使用者綁定信用卡。
// 這是 after：型別用 LogValuer 決定自己被記成什麼，handler 的 ReplaceAttr 依鍵名遮蔽，作為最後一道防線。
package wallet

import (
	"encoding/json"
	"log/slog"
	"maps"
	"net/http"
	"slices"
	"strings"
)

// ANCHOR: types

// redacted 是敏感值在日誌裡的樣子。
const redacted = "[REDACTED]"

// Secret 是密碼、token 這類完全不該出現在日誌裡的字串。
type Secret string

// LogValue 實作 slog.LogValuer：不論值是什麼，日誌裡都只有 [REDACTED]。
func (Secret) LogValue() slog.Value { return slog.StringValue(redacted) }

// Card 是完整的卡號，可以含空格或連字號。
type Card string

// LogValue 實作 slog.LogValuer：只留末四碼給除錯用；不足四位數字時整個遮蔽。
func (c Card) LogValue() slog.Value {
	var digits []rune
	for _, r := range c {
		if r >= '0' && r <= '9' {
			digits = append(digits, r)
		}
	}
	if len(digits) < 4 {
		return slog.StringValue(redacted)
	}
	return slog.GroupValue(slog.String("last4", string(digits[len(digits)-4:])))
}

// BindRequest 是 POST /cards 的請求本文。
type BindRequest struct {
	// UserID 是使用者代號。
	UserID string `json:"user_id"`
	// Password 是再次確認身分用的密碼。
	Password Secret `json:"password"`
	// Card 是要綁定的完整卡號。
	Card Card `json:"card_number"`
}

// LogValue 實作 slog.LogValuer：整個請求的輸出由這裡逐欄決定。
// 之後新增的欄位不會自動進日誌，要有人在這裡明確加上。
func (r BindRequest) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("user_id", r.UserID),
		slog.Any("password", r.Password),
		slog.Any("card", r.Card),
	)
}

// Headers 是請求標頭。記錄它時只輸出允許清單上的值，其他標頭只留名稱。
type Headers http.Header

// LogValue 實作 slog.LogValuer。
func (h Headers) LogValue() slog.Value {
	var attrs []slog.Attr
	for _, name := range slices.Sorted(maps.Keys(h)) {
		switch name {
		case "Content-Type", "User-Agent", "X-Request-Id":
			attrs = append(attrs, slog.String(name, strings.Join(h[name], ",")))
		default:
			attrs = append(attrs, slog.String(name, redacted))
		}
	}
	return slog.GroupValue(attrs...)
}

// ANCHOR_END: types

// ANCHOR: redact

// Redact 是給 slog.HandlerOptions.ReplaceAttr 用的函式：鍵名（不分大小寫）
// 是敏感欄位時，不管值是什麼，一律換成 [REDACTED]。
// 它只看得到屬性的鍵名；包在 slog.Any 裡的 struct 不會被展開。
func Redact(groups []string, a slog.Attr) slog.Attr {
	switch strings.ToLower(a.Key) {
	case "password", "token", "secret", "api_key", "card_number", "cvv",
		"authorization", "cookie", "set-cookie":
		return slog.String(a.Key, redacted)
	}
	return a
}

// ANCHOR_END: redact

// ANCHOR: handler

// NewHandler 回傳處理 POST /cards 的 handler。請求與標頭都記，但交給型別決定怎麼輸出。
func NewHandler(logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req BindRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		logger.Info("bind card", slog.Any("request", req), slog.Any("headers", Headers(r.Header)))
		w.WriteHeader(http.StatusNoContent)
	})
}

// ANCHOR_END: handler
