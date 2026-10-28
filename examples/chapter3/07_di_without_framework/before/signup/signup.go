// Package signup 是常見寫法：相依放在套件層級的變數，在 init 裡接線，Register 直接取用。
package signup

import (
	"errors"
	"fmt"
	"sync"
)

// ErrExists 表示這個信箱已經註冊過。
var ErrExists = errors.New("already registered")

// Mailer 是寄信的能力。
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

// smtpMailer 是示範用的寄信實作：不真的連線，只印出它會寄給誰。
type smtpMailer struct{ addr string }

func (m *smtpMailer) Send(to, subject string) error {
	fmt.Printf("smtp %s: %s mail to %s\n", m.addr, subject, to)
	return nil
}

// ANCHOR: globals

// 套件層級的相依，由 init 接線。
var (
	store  *Store
	mailer Mailer
)

func init() {
	store = NewStore()
	mailer = &smtpMailer{addr: "smtp.example.com:25"}
}

// ANCHOR_END: globals

// ANCHOR: register

// Register 記下 email 並寄出歡迎信。信箱已經註冊過時回傳 ErrExists；
// 信寄不出去時回傳包裝過的錯誤，但註冊已經成立。
func Register(email string) error {
	if !store.Add(email) {
		return ErrExists
	}
	if err := mailer.Send(email, "welcome"); err != nil {
		return fmt.Errorf("send welcome mail to %s: %w", email, err)
	}
	return nil
}

// ANCHOR_END: register
