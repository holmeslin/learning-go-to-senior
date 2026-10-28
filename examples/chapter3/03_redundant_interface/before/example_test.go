package loyalty_test

import (
	"fmt"

	loyalty "github.com/holmeslin/learning-go-to-senior/examples/chapter3/03_redundant_interface/before"
)

// ANCHOR: example

func ExampleNewService_fakeStore() {
	svc := loyalty.NewService(&fakeStore{points: map[string]int{}}, &fakeNotifier{})

	points, err := svc.Earn("ghost", 200)
	fmt.Println(points, err)
	// Output: 20 <nil>
}

func ExampleNewService_realStore() {
	svc := loyalty.NewService(loyalty.NewStore(), &fakeNotifier{})

	points, err := svc.Earn("ghost", 200)
	fmt.Println(points, err)
	// Output: 0 earn: member ghost: not found
}

// ANCHOR_END: example
