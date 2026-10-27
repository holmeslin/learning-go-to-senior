// Package taskapi 是「每一層都 log 再回傳」的常見寫法：store、service、handler 各自記錄錯誤。
package taskapi

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
)

var errConnRefused = errors.New("connection refused")

// Store 是任務儲存。Down 為 true 時，Save 模擬資料庫連不上。
type Store struct {
	Log  *log.Logger
	Down bool
}

// ANCHOR: store

// Save 儲存任務，失敗時先記錄再回傳。
func (s *Store) Save(id int) error {
	if s.Down {
		err := fmt.Errorf("save task %d: %w", id, errConnRefused)
		s.Log.Printf("store: %v", err)
		return err
	}
	return nil
}

// ANCHOR_END: store

// Service 放任務的規則。
type Service struct {
	Log   *log.Logger
	Store *Store
}

// ANCHOR: service

// Complete 完成任務，失敗時先記錄再回傳。
func (s *Service) Complete(id int) error {
	if err := s.Store.Save(id); err != nil {
		err = fmt.Errorf("complete: %w", err)
		s.Log.Printf("service: %v", err)
		return err
	}
	return nil
}

// ANCHOR_END: service

// ANCHOR: handler

// NewHandler 回傳處理 POST /tasks/{id}/complete 的 http.Handler，失敗時記錄並回 500。
func NewHandler(svc *Service, logger *log.Logger) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /tasks/{id}/complete", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			http.Error(w, "bad id", http.StatusBadRequest)
			return
		}
		err = svc.Complete(id)
		if err != nil {
			logger.Printf("handler: %s %s: %v", r.Method, r.URL.Path, err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	return mux
}

// ANCHOR_END: handler
