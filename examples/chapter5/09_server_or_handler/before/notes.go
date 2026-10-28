// Package notes 是一個小小的筆記服務：用 id 讀取與刪除筆記，刪除需要 token。
package notes

import (
	"net/http"
	"sync"
)

// service 保存筆記，getNote 與 deleteNote 是讀取與刪除的 handler 函式。
type service struct {
	token string
	mu    sync.Mutex
	notes map[string]string
}

// newService 建立一個 service，並複製 notes，之後修改傳入的 map 不影響它。
func newService(token string, notes map[string]string) *service {
	s := &service{token: token, notes: map[string]string{}}
	for id, text := range notes {
		s.notes[id] = text
	}
	return s
}

func (s *service) getNote(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	text, ok := s.notes[r.PathValue("id")]
	s.mu.Unlock()
	if !ok {
		http.Error(w, "note not found", http.StatusNotFound)
		return
	}
	w.Write([]byte(text))
}

func (s *service) deleteNote(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	delete(s.notes, r.PathValue("id"))
	s.mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func requireToken(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// NewHandler 回傳組好的 http.Handler：讀取公開，刪除要帶 "Authorization: Bearer <token>"。
// 它會複製 notes，之後修改傳入的 map 不影響 Handler。
func NewHandler(token string, notes map[string]string) http.Handler {
	s := newService(token, notes)
	// ANCHOR: routes
	mux := http.NewServeMux()
	mux.HandleFunc("GET /note/{id}", s.getNote)
	mux.HandleFunc("DELETE /notes/{id}", s.deleteNote)
	return mux
	// ANCHOR_END: routes
}
