package signup

import "sync"

// Store 把活動放在記憶體裡，只供示範：重啟就消失，沒有容量上限；正式環境要放進資料庫。
type Store struct {
	mu       sync.Mutex
	notifier Notifier
	events   map[string]*Event
}

// NewStore 建立空的 Store；它建立的活動都用 n 通知被遞補的人。
func NewStore(n Notifier) *Store {
	return &Store{notifier: n, events: make(map[string]*Event)}
}

// Create 以代號 id 建立名額為 seats 的活動並存起來。代號已存在回傳 ErrEventExists。
func (s *Store) Create(id string, seats int) (*Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.events[id]; ok {
		return nil, ErrEventExists
	}
	ev := NewEvent(seats, s.notifier)
	s.events[id] = ev
	return ev, nil
}

// Get 以代號取出活動；找不到時第二個回傳值是 false。
func (s *Store) Get(id string) (*Event, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ev, ok := s.events[id]
	return ev, ok
}
