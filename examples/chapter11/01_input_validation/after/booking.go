// Package booking 是會議室預約的 handler：在入口限制大小、拒絕未知欄位，
// 把輸入轉成驗證過的 [Booking]，並一次回報所有欄位的錯誤。
package booking

import (
	"errors"
	"net/mail"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ANCHOR: types

// Request 是預約請求的 JSON 內容，還沒有經過驗證。
type Request struct {
	// Name 是預約人的姓名。
	Name string `json:"name"`
	// Email 是預約人的 email。
	Email string `json:"email"`
	// Room 是會議室代號。
	Room string `json:"room"`
	// Guests 是出席人數。
	Guests int `json:"guests"`
}

// FieldError 說明哪一個欄位有什麼問題；Field 是空字串表示問題出在整個請求本文。
type FieldError struct {
	// Field 是 JSON 的欄位名稱。
	Field string `json:"field,omitempty"`
	// Message 是固定的英文訊息，不含使用者的輸入。
	Message string `json:"message"`
}

// Error 回傳「欄位: 訊息」。
func (e *FieldError) Error() string {
	if e.Field == "" {
		return e.Message
	}
	return e.Field + ": " + e.Message
}

// ANCHOR_END: types

// ANCHOR: booking

// Booking 是通過驗證的預約。欄位不匯出，只有 [Request.Validate] 能建出它，
// 所以收到 Booking 的程式不必再檢查一次。
type Booking struct {
	name, email, room string
	guests            int
}

// Name 回傳去掉前後空白的姓名。
func (b Booking) Name() string { return b.name }

// Email 回傳去掉前後空白的 email。
func (b Booking) Email() string { return b.email }

// Room 回傳會議室代號，一定是已知的房間。
func (b Booking) Room() string { return b.room }

// Guests 回傳出席人數，一定在 1 到 20 之間。
func (b Booking) Guests() int { return b.guests }

// Validate 先正規化（去掉前後空白），再檢查正規化後的值，每個有問題的欄位
// 各產生一個 [*FieldError]，用 [errors.Join] 合併，順序依欄位的順序。
func (r Request) Validate() (Booking, error) {
	var errs []error
	bad := func(field, msg string) { errs = append(errs, &FieldError{Field: field, Message: msg}) }

	name := strings.TrimSpace(r.Name)
	switch {
	case name == "":
		bad("name", "required")
	case utf8.RuneCountInString(name) > 50:
		bad("name", "must be at most 50 characters")
	case strings.ContainsFunc(name, unicode.IsControl):
		bad("name", "must not contain control characters")
	}

	email := strings.TrimSpace(r.Email)
	if a, err := mail.ParseAddress(email); err != nil || a.Address != email {
		bad("email", "must be a plain email address")
	}

	room := strings.TrimSpace(r.Room)
	switch room {
	case "a1", "a2", "b1": // 白名單：只收已知的房間
	default:
		bad("room", "unknown room")
	}

	if r.Guests < 1 || r.Guests > 20 {
		bad("guests", "must be between 1 and 20")
	}

	if err := errors.Join(errs...); err != nil {
		return Booking{}, err
	}
	return Booking{name: name, email: email, room: room, guests: r.Guests}, nil
}

// ANCHOR_END: booking
