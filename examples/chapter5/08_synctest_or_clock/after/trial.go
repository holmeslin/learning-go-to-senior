package retry

import "time"

// ANCHOR: trial

// Trial 是一段試用期：到 End 那一天結束（以 End 的時區為準）為止都有效。
type Trial struct {
	End time.Time
	Now func() time.Time // 注入的時鐘；nil 表示用 time.Now
}

// Active 回報試用期現在是否還有效。
func (t Trial) Active() bool {
	now := time.Now
	if t.Now != nil {
		now = t.Now
	}
	y, m, d := t.End.Date()
	return now().Before(time.Date(y, m, d+1, 0, 0, 0, 0, t.End.Location()))
}

// ANCHOR_END: trial
