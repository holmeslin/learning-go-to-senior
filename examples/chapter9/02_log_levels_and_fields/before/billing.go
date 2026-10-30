// Package billing 是一個小小的付款服務：依發票代號扣款，也能批次匯入發票。
// 這是 before：每一層都記錄錯誤，鍵名各寫各的，迴圈裡每一筆都用 Info。
package billing

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
)

// Store 是儲存層。Down 為 true 時，Charge 一律失敗。
type Store struct {
	// Down 為 true 時，Charge 一律回傳錯誤。
	Down bool
	// Logger 是儲存層自己的 logger。
	Logger *slog.Logger
}

// ANCHOR: store

// Charge 對發票扣款。
func (s *Store) Charge(userID, invoiceID string) error {
	if s.Down {
		err := errors.New("connection refused")
		s.Logger.Error("charge failed", "userId", userID, "err", err)
		return err
	}
	return nil
}

// ANCHOR_END: store

// Service 是業務層。
type Service struct {
	// Store 是扣款用的儲存層。
	Store *Store
	// Logger 是業務層自己的 logger。
	Logger *slog.Logger
}

// ANCHOR: service

// Pay 替使用者付清發票。
func (s *Service) Pay(userID, invoiceID string) error {
	if err := s.Store.Charge(userID, invoiceID); err != nil {
		s.Logger.Error("pay failed", "user_id", userID, "invoice_id", invoiceID, "error", err)
		return fmt.Errorf("pay invoice %s: %w", invoiceID, err)
	}
	return nil
}

// ANCHOR_END: service

// NewHandler 回傳處理 POST /invoices/{id}/pay 的 handler，使用者代號取自 X-User-ID 標頭。
func NewHandler(svc *Service, logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	// ANCHOR: handler
	mux.HandleFunc("POST /invoices/{id}/pay", func(w http.ResponseWriter, r *http.Request) {
		uid, inv := r.Header.Get("X-User-ID"), r.PathValue("id")
		if err := svc.Pay(uid, inv); err != nil {
			logger.Error("request failed", "uid", uid, "inv", inv, "err", err)
			http.Error(w, "payment failed", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	// ANCHOR_END: handler
	return mux
}

// ImportInvoices 逐筆匯入發票代號，空字串是壞掉的資料列，回傳匯入的筆數。
func ImportInvoices(logger *slog.Logger, ids []string) int {
	// ANCHOR: import
	imported := 0
	for _, id := range ids {
		if id == "" {
			logger.Info("empty row")
			continue
		}
		logger.Info("imported", "invoiceId", id)
		imported++
	}
	// ANCHOR_END: import
	return imported
}
