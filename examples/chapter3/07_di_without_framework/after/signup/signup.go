// Package signup 的元件都用建構函式收下相依，自己不持有任何套件層級的狀態。
package signup

import (
	"errors"
	"fmt"
	"sync"
)

// ErrExists 表示這個信箱已經註冊過。
var ErrExists = errors.New("already registered")

// Mailer 是 Service 需要的寄信能力。介面宣告在使用它的這一端，只含 Service 用到的方法。
type Mailer interface {
	// Send 寄一封主旨為 subject 的信給 to。
	Send(to, subject string) error
}

// Store 以記憶體保存已註冊的信箱。
type Store struct {
	mu    sync.Mutex
	users map[string]bool
}

// NewStore 回傳一個空的 Store。
func NewStore() *Store {
	return &Store{users: map[string]bool{}}
}

// Add 記下 email；已經存在時回傳 false。
func (s *Store) Add(email string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.users[email] {
		return false
	}
	s.users[email] = true
	return true
}

// SMTPMailer 是示範用的寄信實作：不真的連線，只印出它會寄給誰。
type SMTPMailer struct{ addr string }

// NewSMTPMailer 回傳寄往 addr 這台 SMTP 伺服器的 SMTPMailer。
func NewSMTPMailer(addr string) *SMTPMailer {
	return &SMTPMailer{addr: addr}
}

// Send 實作 Mailer。
func (m *SMTPMailer) Send(to, subject string) error {
	fmt.Printf("smtp %s: %s mail to %s\n", m.addr, subject, to)
	return nil
}

// ANCHOR: service

// Service 處理註冊。它需要的 Store 與 Mailer 都由建構函式收進來。
type Service struct {
	store  *Store
	mailer Mailer
}

// NewService 回傳使用 store 與 mailer 的 Service。
func NewService(store *Store, mailer Mailer) *Service {
	return &Service{store: store, mailer: mailer}
}

// ANCHOR_END: service

// ANCHOR: register

// Register 記下 email 並寄出歡迎信。信箱已經註冊過時回傳 ErrExists；
// 信寄不出去時回傳包裝過的錯誤，但註冊已經成立。
func (s *Service) Register(email string) error {
	if !s.store.Add(email) {
		return ErrExists
	}
	if err := s.mailer.Send(email, "welcome"); err != nil {
		return fmt.Errorf("send welcome mail to %s: %w", email, err)
	}
	return nil
}

// ANCHOR_END: register
