package signup

import "slices"

// ANCHOR: limit

// MaxWaitlist 是每場活動候補名單的人數上限。
//
// 這是匯出的可變套件變數，是刻意保留的問題（review 意見 R5）：所有活動共用同一個值，
// 任何套件、任何時候都能改它，測試之間也會互相影響。
var MaxWaitlist = 50

// ANCHOR_END: limit

// Waitlisted 回傳候補名單，依排隊順序。
func (e *Event) Waitlisted() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return slices.Clone(e.waitlist)
}

// ANCHOR: promote

// PromoteNext 在有空位時，把候補名單的第一位遞補為正式名單並通知他，
// 回傳被遞補的人；沒有空位或沒有人在候補時回傳 "" 與 false。
// 取消之後，呼叫端要記得呼叫它。
func (e *Event) PromoteNext() (string, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	cf, wl := e.confirmed, e.waitlist
	if len(cf) >= e.seats || len(wl) == 0 {
		return "", false
	}
	nx := wl[0]
	e.waitlist = slices.Delete(wl, 0, 1)
	e.confirmed = append(cf, nx)
	e.notifier.Notify(nx)
	return nx, true
}

// ANCHOR_END: promote
