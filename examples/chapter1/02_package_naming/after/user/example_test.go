package user_test

import (
	"errors"
	"fmt"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter1/02_package_naming/after/user"
)

// ANCHOR: example
func ExampleService_Register() {
	svc := user.NewService()

	u, err := svc.Register("Holmes Lin")
	fmt.Println(u.Slug, err)

	_, err = svc.Register("Admin")
	fmt.Println(errors.Is(err, user.ErrReserved))
	// Output:
	// holmes-lin <nil>
	// true
}

// ANCHOR_END: example
