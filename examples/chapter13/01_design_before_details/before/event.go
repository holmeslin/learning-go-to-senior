// Package signup 是虛構的「活動報名平台」的核心：有名額上限的活動、
// 報名與取消、候補名單、放在記憶體裡的儲存，以及 HTTP handler。
//
// 這是 review 前的版本：作者送出「候補名單」PR 時的樣子。
// 其中的問題是刻意保留的（review 意見 R1 到 R7，見本章各集），不要照用。
package signup

import (
	"slices"
	"sync"
)

// Event 是一場有名額上限的活動。零值不可用，請用 NewEvent 建立。
type Event struct {
	mu        sync.Mutex
	seats     int
	notifier  Notifier
	confirmed []string
	waitlist  []string
}

// NewEvent 建立名額為 seats 的活動；有人從候補名單遞補時，用 n 通知他。
func NewEvent(seats int, n Notifier) *Event {
	return &Event{seats: seats, notifier: n}
}

// Seats 回傳名額上限。
func (e *Event) Seats() int {
	return e.seats
}

// ANCHOR: register

// Register 為 who 報名。名額已滿時排進候補名單；候補名單也滿了回傳 ErrFull。
// 已經報名過回傳 ErrAlreadyRegistered，名稱是空的回傳 ErrEmptyName。
func (e *Event) Register(who string) error {
	who, err := normalize(who)
	if err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if slices.Contains(e.confirmed, who) {
		return ErrAlreadyRegistered
	}
	if len(e.confirmed) >= e.seats {
		if len(e.waitlist) >= MaxWaitlist {
			return ErrFull
		}
		e.waitlist = append(e.waitlist, who)
		return nil
	}
	e.confirmed = append(e.confirmed, who)
	return nil
}

// ANCHOR_END: register

// ANCHOR: cancel

// Cancel 取消 who 的報名，正式名單與候補名單都找。沒有這個人的報名紀錄回傳 ErrNotRegistered。
// 它只移除這個人；空出來的名額由 PromoteNext 遞補。
func (e *Event) Cancel(who string) error {
	who, err := normalize(who)
	if err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if i := slices.Index(e.confirmed, who); i >= 0 {
		e.confirmed = slices.Delete(e.confirmed, i, i+1)
		return nil
	}
	if i := slices.Index(e.waitlist, who); i >= 0 {
		e.waitlist = slices.Delete(e.waitlist, i, i+1)
		return nil
	}
	return ErrNotRegistered
}

// ANCHOR_END: cancel

// Confirmed 回傳已確定報名的人，依報名順序。
func (e *Event) Confirmed() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return slices.Clone(e.confirmed)
}
