package signup_test

import (
	signup "github.com/holmeslin/learning-go-to-senior/examples/chapter13/01_design_before_details/after"
)

// ANCHOR: newevent

// newEvent 建立測試用的活動；候補名單上限固定是 10，多數測試只關心名額與通知。
func newEvent(seats int, n signup.Notifier) *signup.Event {
	return signup.NewEvent(seats, n, 10)
}

// ANCHOR_END: newevent
