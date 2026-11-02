package signup_test

import (
	"fmt"

	signup "github.com/holmeslin/learning-go-to-senior/examples/chapter13/01_design_before_details/base"
)

// 名額 2 的活動：前兩位報名成功，第三位被拒絕。
func ExampleEvent() {
	ev := signup.NewEvent(2)
	for _, who := range []string{"u1", "u2", "u3"} {
		fmt.Println(who, ev.Register(who))
	}
	fmt.Println(ev.Attendees())
	// Output:
	// u1 <nil>
	// u2 <nil>
	// u3 signup: event is full
	// [u1 u2]
}
