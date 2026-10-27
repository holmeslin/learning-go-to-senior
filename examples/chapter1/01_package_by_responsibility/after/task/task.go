// Package task 管理待辦任務：資料、規則與儲存都在這一個套件裡。
package task

import (
	"errors"
	"sync"
)

// ErrNotFound 表示找不到指定 ID 的任務。
var ErrNotFound = errors.New("task: not found")

// Task 是一筆待辦任務。
type Task struct {
	ID    int
	Title string
	Done  bool
}

// Store 是放在記憶體裡的任務儲存，可以同時被多個 goroutine 使用。
type Store struct {
	mu     sync.Mutex
	nextID int
	tasks  map[int]*Task
}

// NewStore 建立一個空的 Store。
func NewStore() *Store {
	return &Store{tasks: make(map[int]*Task)}
}

// Add 新增一筆任務並回傳它。
func (s *Store) Add(title string) Task {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	t := &Task{ID: s.nextID, Title: title}
	s.tasks[t.ID] = t
	return *t
}

// ANCHOR: complete
// Complete 把任務標記為完成，找不到時回傳 ErrNotFound。
func (s *Store) Complete(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tasks[id]
	if !ok {
		return ErrNotFound
	}
	t.Done = true
	return nil
}

// ANCHOR_END: complete

// Get 回傳指定 ID 任務的複本。
func (s *Store) Get(id int) (Task, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tasks[id]
	if !ok {
		return Task{}, false
	}
	return *t, true
}
