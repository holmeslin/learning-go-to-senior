// Package loyalty 計算會員點數：消費累積點數，點數跨過門檻時通知會員。
// 只有 Notifier 是介面，因為通知會送到系統外面。
package loyalty

import (
	"errors"
	"fmt"
	"sync"
)

// GoldPoints 是升級成金卡會員所需的點數。
const GoldPoints = 100

// ErrNotFound 表示找不到指定 ID 的會員。
var ErrNotFound = errors.New("not found")

// Member 是一位會員與他目前的點數。
type Member struct {
	ID     string
	Points int
}

// Store 是放在記憶體裡的會員儲存，可以同時被多個 goroutine 使用。
type Store struct {
	mu      sync.Mutex
	members map[string]Member
}

// NewStore 建立一個 Store，裡面已經有 members 這幾位會員。
func NewStore(members ...Member) *Store {
	s := &Store{members: make(map[string]Member, len(members))}
	for _, m := range members {
		s.members[m.ID] = m
	}
	return s
}

// Get 回傳指定 ID 的會員，找不到時回傳 ErrNotFound。
func (s *Store) Get(id string) (Member, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.members[id]
	if !ok {
		return Member{}, fmt.Errorf("member %s: %w", id, ErrNotFound)
	}
	return m, nil
}

// Add 替指定的會員加上 points 點，回傳加完之後的會員；找不到時回傳 ErrNotFound。
func (s *Store) Add(id string, points int) (Member, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.members[id]
	if !ok {
		return Member{}, fmt.Errorf("member %s: %w", id, ErrNotFound)
	}
	m.Points += points
	s.members[id] = m
	return m, nil
}

// ANCHOR: service

// Notifier 把訊息送出系統外（寄信、呼叫 webhook……），是這個套件與外面世界的邊界。
type Notifier interface {
	Notify(id, msg string) error
}

// Service 計算點數。它直接用 *Store，中間沒有介面。
type Service struct {
	store    *Store
	notifier Notifier
}

// NewService 建立點數服務。
func NewService(store *Store, notifier Notifier) *Service {
	return &Service{store: store, notifier: notifier}
}

// Earn 依消費金額（每 10 元 1 點）替會員加點，回傳加完之後的點數。
// 點數跨過 GoldPoints 時會通知會員；通知失敗時點數已經入帳，回傳的點數是入帳後的，error 是通知的錯誤。
func (s *Service) Earn(id string, spent int) (int, error) {
	if spent <= 0 {
		return 0, fmt.Errorf("earn: spent must be positive, got %d", spent)
	}
	earned := spent / 10
	m, err := s.store.Add(id, earned)
	if err != nil {
		return 0, fmt.Errorf("earn: %w", err)
	}
	if m.Points-earned < GoldPoints && m.Points >= GoldPoints {
		err = s.notifier.Notify(id, "gold member")
		if err != nil {
			return m.Points, fmt.Errorf("earn: notify %s: %w", id, err)
		}
	}
	return m.Points, nil
}

// Balance 回傳會員目前的點數，找不到會員時回傳 ErrNotFound。
func (s *Service) Balance(id string) (int, error) {
	m, err := s.store.Get(id)
	if err != nil {
		return 0, fmt.Errorf("balance: %w", err)
	}
	return m.Points, nil
}

// ANCHOR_END: service
