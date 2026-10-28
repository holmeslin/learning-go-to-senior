// Package loyalty 計算會員點數。這是「每個元件都先配一個介面」的寫法。
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

// ANCHOR: interfaces

// Service 是點數服務。
type Service interface {
	Earn(id string, spent int) (int, error)
	Balance(id string) (int, error)
}

// Store 是會員儲存。
type Store interface {
	Get(id string) (Member, error)
	Add(id string, points int) (Member, error)
}

// Notifier 把訊息送出系統外。
type Notifier interface {
	Notify(id, msg string) error
}

// ANCHOR_END: interfaces

// ANCHOR: service

type service struct {
	store    Store
	notifier Notifier
}

// NewService 建立點數服務。
func NewService(store Store, notifier Notifier) Service {
	return &service{store: store, notifier: notifier}
}

func (s *service) Earn(id string, spent int) (int, error) {
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

func (s *service) Balance(id string) (int, error) {
	m, err := s.store.Get(id)
	if err != nil {
		return 0, fmt.Errorf("balance: %w", err)
	}
	return m.Points, nil
}

// ANCHOR_END: service

type memStore struct {
	mu      sync.Mutex
	members map[string]Member
}

// NewStore 建立一個 Store，裡面已經有 members 這幾位會員。
func NewStore(members ...Member) Store {
	s := &memStore{members: make(map[string]Member, len(members))}
	for _, m := range members {
		s.members[m.ID] = m
	}
	return s
}

func (s *memStore) Get(id string) (Member, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.members[id]
	if !ok {
		return Member{}, fmt.Errorf("member %s: %w", id, ErrNotFound)
	}
	return m, nil
}

func (s *memStore) Add(id string, points int) (Member, error) {
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
