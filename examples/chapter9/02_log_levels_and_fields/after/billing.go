// Package billing 是一個小小的付款服務：依發票代號扣款，也能批次匯入發票。
// 這是 after：錯誤只在處理的那一層記錄，鍵名集中定義，請求範圍的欄位用 With 帶入。
package billing

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// Store 是儲存層，它不記錄日誌。Down 為 true 時，Charge 一律失敗。
type Store struct {
	// Down 為 true 時，Charge 一律回傳錯誤。
	Down bool
}

// Charge 對發票扣款。
func (s *Store) Charge(invoiceID string) error {
	if s.Down {
		return errors.New("connection refused")
	}
	return nil
}

// Service 是業務層。
type Service struct {
	// Store 是扣款用的儲存層。
	Store *Store
}

// ANCHOR: service

// Pay 付清發票。log 已經帶著請求範圍的欄位；失敗時只加上下文就回傳，不記 Error。
func (s *Service) Pay(log *slog.Logger, invoiceID string) error {
	log.Debug("charge invoice")
	if err := s.Store.Charge(invoiceID); err != nil {
		return fmt.Errorf("pay invoice %s: %w", invoiceID, err)
	}
	return nil
}

// ANCHOR_END: service

// NewHandler 回傳處理 POST /invoices/{id}/pay 的 handler。
// 請求代號與使用者代號取自 X-Request-ID、X-User-ID 標頭；handler 是邊界，錯誤在這裡記錄一次。
func NewHandler(svc *Service, logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	// ANCHOR: handler
	mux.HandleFunc("POST /invoices/{id}/pay", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		invoiceID := r.PathValue("id")
		log := logger.With(
			slog.String(KeyRequestID, r.Header.Get("X-Request-ID")),
			slog.String(KeyUserID, r.Header.Get("X-User-ID")),
			slog.String(KeyInvoiceID, invoiceID),
		)
		if err := svc.Pay(log, invoiceID); err != nil {
			log.Error("pay invoice failed", slog.Any(KeyError, err), Duration(time.Since(start)))
			http.Error(w, "payment failed", http.StatusInternalServerError)
			return
		}
		log.Info("invoice paid", Duration(time.Since(start)))
		w.WriteHeader(http.StatusNoContent)
	})
	// ANCHOR_END: handler
	return mux
}

// ImportInvoices 逐筆匯入發票代號，空字串是壞掉的資料列，回傳匯入的筆數。
// 每一筆只記 Debug；整批結束記一行摘要，有跳過的資料列時是 Warn，否則是 Info。
func ImportInvoices(logger *slog.Logger, ids []string) int {
	// ANCHOR: import
	imported, skipped := 0, 0
	for _, id := range ids {
		if id == "" {
			skipped++
			logger.Debug("skip empty row")
			continue
		}
		imported++
		logger.Debug("import row", slog.String(KeyInvoiceID, id))
	}
	level := slog.LevelInfo
	if skipped > 0 {
		level = slog.LevelWarn
	}
	logger.LogAttrs(context.Background(), level, "import finished",
		slog.Int(KeyImported, imported), slog.Int(KeySkipped, skipped))
	// ANCHOR_END: import
	return imported
}
