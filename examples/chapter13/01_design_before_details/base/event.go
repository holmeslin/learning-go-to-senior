// Package signup 是虛構的「活動報名平台」的核心：有名額上限的活動、
// 報名與取消、放在記憶體裡的儲存，以及 HTTP handler。
//
// 這是 base：候補名單 PR 送出之前的主幹。
package signup

import (
	"slices"
	"strings"
	"sync"
)

// Event 是一場有名額上限的活動。零值不可用，請用 NewEvent 建立。
type Event struct {
	mu        sync.Mutex
	capacity  int
	attendees []string
}

// NewEvent 建立名額為 capacity 的活動。
func NewEvent(capacity int) *Event {
	return &Event{capacity: capacity}
}

// Capacity 回傳名額上限。
func (e *Event) Capacity() int {
	return e.capacity
}

// Register 為 who 報名。名額已滿回傳 ErrFull，已經報名過回傳 ErrAlreadyRegistered，
// 名稱是空的回傳 ErrEmptyName。
func (e *Event) Register(who string) error {
	who, err := normalize(who)
	if err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if slices.Contains(e.attendees, who) {
		return ErrAlreadyRegistered
	}
	if len(e.attendees) >= e.capacity {
		return ErrFull
	}
	e.attendees = append(e.attendees, who)
	return nil
}

// Cancel 取消 who 的報名。沒有這個人的報名紀錄回傳 ErrNotRegistered。
func (e *Event) Cancel(who string) error {
	who, err := normalize(who)
	if err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	i := slices.Index(e.attendees, who)
	if i < 0 {
		return ErrNotRegistered
	}
	e.attendees = slices.Delete(e.attendees, i, i+1)
	return nil
}

// Attendees 回傳已報名的人，依報名順序。
func (e *Event) Attendees() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return slices.Clone(e.attendees)
}

// normalize 去掉名稱前後的空白；結果是空的回傳 ErrEmptyName。
func normalize(who string) (string, error) {
	who = strings.TrimSpace(who)
	if who == "" {
		return "", ErrEmptyName
	}
	return who, nil
}
