package coupon_test

import (
	"fmt"

	coupon "github.com/holmeslin/learning-go-to-senior/examples/chapter5/02_table_driven_boundary/after"
)

func ExampleBook_Redeem() {
	b := coupon.NewBook(
		coupon.Coupon{Code: "SAVE10", Percent: 10},
		coupon.Coupon{Code: "WELCOME20", Percent: 20, MinSpend: 500, SingleUse: true},
		coupon.Coupon{Code: "VIP5", Percent: 5, MinSpend: 2000},
	)
	fmt.Println(b.Redeem("save10", 995))
	fmt.Println(b.Redeem("VIP5", 1999))
	fmt.Println(b.Redeem("WELCOME20", 500))
	fmt.Println(b.Redeem("WELCOME20", 500))
	fmt.Println(b.Redeem("NOPE", 100))
	// Output:
	// 896 <nil>
	// 0 redeem VIP5: below minimum spend of 2000
	// 400 <nil>
	// 0 redeem WELCOME20: used up
	// 0 redeem NOPE: unknown code
}
