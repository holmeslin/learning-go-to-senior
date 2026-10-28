// Package notes 是一個小小的筆記服務：用 id 讀取與刪除筆記，刪除需要 token。
package notes

import (
	"net/http"
	"sync"
)

// Server 保存筆記，並提供讀取與刪除的 HTTP 端點。
type Server struct {
	token string
	mu    sync.Mutex
	notes map[string]string
}

// NewServer 建立一個 Server。刪除請求必須帶 "Authorization: Bearer <token>"。
// 它會複製 notes，之後修改傳入的 map 不影響 Server。
func NewServer(token string, notes map[string]string) *Server {
	s := &Server{token: token, notes: map[string]string{}}
	for id, text := range notes {
		s.notes[id] = text
	}
	return s
}

func (s *Server) getNote(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	text, ok := s.notes[r.PathValue("id")]
	s.mu.Unlock()
	if !ok {
		http.Error(w, "note not found", http.StatusNotFound)
		return
	}
	w.Write([]byte(text))
}

func (s *Server) deleteNote(w http.ResponseWriter, r *http.Request) {
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

// Handler 回傳組好的路由：讀取公開，刪除要先通過 token 檢查。
func (s *Server) Handler() http.Handler {
	// ANCHOR: routes
	mux := http.NewServeMux()
	mux.HandleFunc("GET /notes/{id}", s.getNote)
	mux.Handle("DELETE /notes/{id}", requireToken(s.token, http.HandlerFunc(s.deleteNote)))
	return mux
	// ANCHOR_END: routes
}
