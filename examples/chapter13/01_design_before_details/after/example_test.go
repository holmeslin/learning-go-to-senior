package signup_test

import (
	"fmt"

	signup "github.com/holmeslin/learning-go-to-senior/examples/chapter13/01_design_before_details/after"
)

// 名額 2 的活動：前兩位報名成功，第三位排進候補，而且呼叫端看得出來。
func ExampleEvent() {
	ev := signup.NewEvent(2, &fakeNotifier{}, 10)
	for _, who := range []string{"u1", "u2", "u3"} {
		fmt.Println(who, ev.Register(who))
	}
	fmt.Println(ev.Confirmed(), ev.Waitlisted())
	// Output:
	// u1 <nil>
	// u2 <nil>
	// u3 signup: event is full, placed on the waitlist
	// [u1 u2] [u3]
}
