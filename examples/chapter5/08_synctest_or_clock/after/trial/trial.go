// Package trial 判斷試用期是否還有效；時鐘由呼叫端注入，所以測試可以指定任何一個時間點。
package trial

import "time"

// ANCHOR: trial

// Period 是一段試用期：到 End 那一天結束（以 End 的時區為準）為止都有效。
type Period struct {
	End time.Time
	Now func() time.Time // 注入的時鐘；nil 表示用 time.Now
}

// Active 回報試用期現在是否還有效。
func (p Period) Active() bool {
	now := time.Now
	if p.Now != nil {
		now = p.Now
	}
	y, m, d := p.End.Date()
	return now().Before(time.Date(y, m, d+1, 0, 0, 0, 0, p.End.Location()))
}

// ANCHOR_END: trial
