// Package payment 是較好的做法：POST /charges 用冪等鍵（Idempotency-Key）讓重複的請求只扣一次款。
package payment

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

// Charge 是一筆扣款紀錄。
type Charge struct {
	ID    string `json:"id"`
	Order string `json:"order"`
	Cents int    `json:"cents"`
}

// Ledger 是記憶體版的帳本，只適合示範；正式環境寫進資料庫。
type Ledger struct {
	mu      sync.Mutex
	charges []Charge
}

// Add 記一筆扣款，回傳帶有編號的紀錄。
func (l *Ledger) Add(order string, cents int) Charge {
	l.mu.Lock()
	defer l.mu.Unlock()
	c := Charge{ID: fmt.Sprintf("ch_%d", len(l.charges)+1), Order: order, Cents: cents}
	l.charges = append(l.charges, c)
	return c
}

// Charges 回傳目前所有的扣款紀錄（複本）。
func (l *Ledger) Charges() []Charge {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]Charge(nil), l.charges...)
}

// Handler 回傳處理 POST /charges 的 handler：每個請求扣一次款，回 201 與扣款紀錄。
// 它本身不防重複，用 Store.Wrap 包起來才有冪等。
func Handler(l *Ledger) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /charges", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Order string `json:"order"`
			Cents int    `json:"cents"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Cents <= 0 {
			http.Error(w, "invalid charge", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(l.Add(req.Order, req.Cents))
	})
	return mux
}
