package booking_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	booking "github.com/holmeslin/learning-go-to-senior/examples/chapter11/01_input_validation/before"
)

// ANCHOR: example

func ExampleNewHandler() {
	h := booking.NewHandler(func(booking.Request) {})
	for _, a := range booking.Attacks() {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/bookings", strings.NewReader(a.Body)))
		fmt.Println(a.Name, rec.Code)
	}
	// Output:
	// oversized_body 201
	// unknown_field 201
	// trailing_value 201
	// guests_out_of_range 201
	// unknown_room 201
	// bad_email 201
	// blank_name 201
	// control_character 201
}

// ANCHOR_END: example

// ANCHOR: leak

func ExampleNewHandler_leak() {
	h := booking.NewHandler(func(booking.Request) {})
	body := `{"name":"Andy","email":"andy@example.com","room":"a1","guests":"4"}`
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/bookings", strings.NewReader(body)))
	fmt.Print(rec.Code, " ", rec.Body)
	// Output:
	// 400 json: cannot unmarshal string into Go struct field Request.guests of type int
}

// ANCHOR_END: leak
