// Package signup 是虛構的「活動報名平台」的核心：有名額上限的活動、
// 報名與取消、候補名單、放在記憶體裡的儲存，以及 HTTP handler。
//
// 這是 review 之後的版本：候補名單 PR 依 review 意見 R1 到 R7 修改的結果。
package signup

import (
	"slices"
	"sync"
)

// Event 是一場有名額上限的活動。零值不可用，請用 NewEvent 建立。
type Event struct {
	mu            sync.Mutex
	seats         int
	waitlistLimit int
	notifier      Notifier
	confirmed     []string
	waitlist      []string
}

// NewEvent 建立名額為 seats、候補名單最多 waitlistLimit 人的活動；
// 有人從候補名單遞補時，用 n 通知他。
func NewEvent(seats int, n Notifier, waitlistLimit int) *Event {
	return &Event{seats: seats, waitlistLimit: waitlistLimit, notifier: n}
}

// Seats 回傳名額上限。
func (e *Event) Seats() int {
	return e.seats
}

// Register 為 who 報名。名額已滿時排進候補名單，並回傳 ErrWaitlisted；
// 候補名單也滿了回傳 ErrFull。已經報名過（正式或候補）回傳 ErrAlreadyRegistered，
// 名稱是空的回傳 ErrEmptyName。
func (e *Event) Register(who string) error {
	who, err := normalize(who)
	if err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if slices.Contains(e.confirmed, who) || slices.Contains(e.waitlist, who) {
		return ErrAlreadyRegistered
	}
	if len(e.confirmed) < e.seats {
		e.confirmed = append(e.confirmed, who)
		return nil
	}
	if len(e.waitlist) >= e.waitlistLimit {
		return ErrFull
	}
	e.waitlist = append(e.waitlist, who)
	return ErrWaitlisted
}

// ANCHOR: cancel

// Cancel 取消 who 的報名，正式名單與候補名單都找。沒有這個人的報名紀錄回傳 ErrNotRegistered。
// 取消的是正式名單的人而候補名單有人時，第一位自動遞補並收到通知；
// 不論從哪裡呼叫 Cancel，都是這個行為。
func (e *Event) Cancel(who string) error {
	who, err := normalize(who)
	if err != nil {
		return err
	}
	promoted, err := e.remove(who)
	if err != nil {
		return err
	}
	if promoted != "" {
		e.notifier.Notify(promoted) // 鎖已經放開：通知再慢，也不會擋住其他報名與取消
	}
	return nil
}

// remove 在鎖內移除 who，並決定誰遞補；回傳被遞補的人，沒有人遞補時回傳 ""。
func (e *Event) remove(who string) (promoted string, err error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if i := slices.Index(e.confirmed, who); i >= 0 {
		e.confirmed = slices.Delete(e.confirmed, i, i+1)
		if len(e.waitlist) == 0 {
			return "", nil
		}
		next := e.waitlist[0]
		e.waitlist = slices.Delete(e.waitlist, 0, 1)
		e.confirmed = append(e.confirmed, next)
		return next, nil
	}
	if i := slices.Index(e.waitlist, who); i >= 0 {
		e.waitlist = slices.Delete(e.waitlist, i, i+1)
		return "", nil
	}
	return "", ErrNotRegistered
}

// ANCHOR_END: cancel

// Confirmed 回傳已確定報名的人，依報名順序。
func (e *Event) Confirmed() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return slices.Clone(e.confirmed)
}
