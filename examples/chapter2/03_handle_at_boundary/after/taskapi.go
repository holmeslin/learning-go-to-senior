// Package taskapi 只在邊界處理錯誤：store 與 service 加上上下文就回傳，handler 與 worker 記錄一次。
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
	Down bool
}

// ANCHOR: store

// Save 儲存任務，失敗時加上任務 ID 再回傳。
func (s *Store) Save(id int) error {
	if s.Down {
		return fmt.Errorf("save task %d: %w", id, errConnRefused)
	}
	return nil
}

// ANCHOR_END: store

// Service 放任務的規則。
type Service struct {
	Store *Store
}

// ANCHOR: service

// Complete 完成任務，失敗時加上自己的上下文再回傳。
func (s *Service) Complete(id int) error {
	if err := s.Store.Save(id); err != nil {
		return fmt.Errorf("complete: %w", err)
	}
	return nil
}

// ANCHOR_END: service

// ANCHOR: handler

// NewHandler 回傳處理 POST /tasks/{id}/complete 的 http.Handler，失敗時記錄一次並回 500。
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

// ANCHOR: runworker

// RunWorker 依序完成 ids 裡的任務，直到 channel 關閉。
// 它通常跑在自己的 goroutine 裡，沒有呼叫端接得到錯誤，所以在這裡記錄一次。
func RunWorker(ids <-chan int, svc *Service, logger *log.Logger) {
	for id := range ids {
		if err := svc.Complete(id); err != nil {
			logger.Printf("worker: task %d: %v", id, err)
		}
	}
}

// ANCHOR_END: runworker
