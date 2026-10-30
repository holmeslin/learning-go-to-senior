package booking_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	booking "github.com/holmeslin/learning-go-to-senior/examples/chapter11/01_input_validation/after"
)

// ANCHOR: example

func Example() {
	h := booking.NewHandler(func(booking.Booking) {})
	body := `{"name":"","email":"nope","room":"x9","guests":100000}`
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/bookings", strings.NewReader(body)))
	fmt.Print(rec.Code, " ", rec.Body)
	// Output:
	// 400 {"errors":[{"field":"name","message":"required"},{"field":"email","message":"must be a plain email address"},{"field":"room","message":"unknown room"},{"field":"guests","message":"must be between 1 and 20"}]}
}

// ANCHOR_END: example
