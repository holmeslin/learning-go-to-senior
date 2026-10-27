package app_test

import (
	"fmt"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter1/02_package_naming/after/app"
)

func ExampleWelcome() {
	msg, err := app.Welcome("Holmes Lin", "寫 Go 的人，缺的是取捨")
	fmt.Println(msg, err)
	// Output: /u/holmes-lin 寫 Go 的人，… <nil>
}
