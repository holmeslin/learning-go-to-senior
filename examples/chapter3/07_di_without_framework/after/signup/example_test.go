package signup_test

import "fmt"

// ANCHOR: example

func ExampleService_Register() {
	svc, mailer := newService()

	fmt.Println(svc.Register("ann@example.com"))
	fmt.Println(mailer.sent)
	// Output:
	// <nil>
	// [ann@example.com]
}

func ExampleService_Register_again() {
	// 另一個測試也註冊 ann@example.com；它自己組了一張新的圖，不受上一個影響。
	svc, mailer := newService()

	fmt.Println(svc.Register("ann@example.com"))
	fmt.Println(svc.Register("ann@example.com"))
	fmt.Println(mailer.sent)
	// Output:
	// <nil>
	// already registered
	// [ann@example.com]
}

// ANCHOR_END: example
