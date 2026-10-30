// Package booking 是會議室預約的 handler。
//
// 這是刻意示範的漏洞，不要照用：它直接解碼進 struct，只檢查必填，
// 超大的本文、未知的欄位與超出範圍的值都照單全收。
package booking

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// ANCHOR: handler

// Request 是預約請求的 JSON 內容。
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

// NewHandler 回傳處理預約的 handler：解碼、檢查必填，然後交給 save。
func NewHandler(save func(Request)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req.Name == "" || req.Email == "" || req.Room == "" {
			http.Error(w, "name, email and room are required", http.StatusBadRequest)
			return
		}
		save(req)
		w.WriteHeader(http.StatusCreated)
	})
}

// ANCHOR_END: handler

// ANCHOR: attacks

// Attack 是一筆惡意輸入：Name 是識別名稱，Body 是原封不動送出的請求本文。
type Attack struct {
	// Name 是這筆輸入的識別名稱，對應「問題在哪」表格的第一欄。
	Name string
	// Body 是原封不動送出的請求本文。
	Body string
}

func request(name, email, room string, guests int) string {
	return fmt.Sprintf(`{"name":%q,"email":%q,"room":%q,"guests":%d}`, name, email, room, guests)
}

// Attacks 回傳「問題在哪」表格裡的每一筆惡意輸入，每次呼叫都產生新的切片。
// 匯出它是為了讓 after 的測試用同一份表格；它不是這個套件的功能。
func Attacks() []Attack {
	const email = "andy@example.com"
	return []Attack{
		{"oversized_body", request(strings.Repeat("a", 2<<20), email, "a1", 2)},
		{"unknown_field", strings.TrimSuffix(request("Andy", email, "a1", 2), "}") + `,"admin":true}`},
		{"trailing_value", request("Andy", email, "a1", 2) + `{"name":"Eve"}`},
		{"guests_out_of_range", request("Andy", email, "a1", 100000)},
		{"unknown_room", request("Andy", email, "x9", 2)},
		{"bad_email", request("Andy", "not-an-email", "a1", 2)},
		{"blank_name", request("   ", email, "a1", 2)},
		{"control_character", request("Andy\r\nX-Admin: 1", email, "a1", 2)},
	}
}

// ANCHOR_END: attacks
