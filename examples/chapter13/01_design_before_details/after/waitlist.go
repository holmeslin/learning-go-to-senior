package signup

import "slices"

// Waitlisted 回傳候補名單，依排隊順序。
func (e *Event) Waitlisted() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return slices.Clone(e.waitlist)
}
