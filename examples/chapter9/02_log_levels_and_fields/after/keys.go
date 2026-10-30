package billing

import (
	"log/slog"
	"time"
)

// ANCHOR: keys

// 團隊共用的日誌鍵名：全部 snake_case，單位寫在鍵名裡。
const (
	// KeyRequestID 是請求代號，同一個請求的所有日誌都帶它。
	KeyRequestID = "request_id"
	// KeyUserID 是使用者代號。
	KeyUserID = "user_id"
	// KeyInvoiceID 是發票代號。
	KeyInvoiceID = "invoice_id"
	// KeyError 是錯誤。
	KeyError = "error"
	// KeyDuration 是耗時，單位固定是毫秒。
	KeyDuration = "duration_ms"
	// KeyImported 是批次裡成功匯入的筆數。
	KeyImported = "imported"
	// KeySkipped 是批次裡跳過的筆數。
	KeySkipped = "skipped"
)

// Duration 回傳以毫秒記錄的耗時欄位，讓每個服務的耗時都是同一個鍵、同一個單位。
func Duration(d time.Duration) slog.Attr {
	return slog.Int64(KeyDuration, d.Milliseconds())
}

// ANCHOR_END: keys
