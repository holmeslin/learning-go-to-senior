package booking_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	booking "github.com/holmeslin/learning-go-to-senior/examples/chapter11/01_input_validation/after"
	before "github.com/holmeslin/learning-go-to-senior/examples/chapter11/01_input_validation/before"
)

// ANCHOR: helpers

// post 把 body 送進 handler，回傳回應、解析好的錯誤清單，以及 save 收到的資料。
func post(t *testing.T, body io.Reader) (*httptest.ResponseRecorder, []booking.FieldError, []booking.Booking) {
	t.Helper()
	var saved []booking.Booking
	h := booking.NewHandler(func(b booking.Booking) { saved = append(saved, b) })
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/bookings", body))
	var resp struct{ Errors []booking.FieldError }
	if rec.Code != http.StatusCreated {
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("error response is not JSON: %v\n%s", err, rec.Body)
		}
	}
	return rec, resp.Errors, saved
}

func fields(errs []booking.FieldError) []string {
	var got []string
	for _, e := range errs {
		got = append(got, e.Field)
	}
	return got
}

// ANCHOR_END: helpers

// ANCHOR: attacks

// 同一份惡意輸入表格：before 全部接受，after 各自以哪個狀態碼、哪些欄位拒絕。
var want = map[string]struct {
	status int
	fields []string
}{
	"oversized_body":      {http.StatusRequestEntityTooLarge, []string{""}},
	"unknown_field":       {http.StatusBadRequest, []string{""}},
	"trailing_value":      {http.StatusBadRequest, []string{""}},
	"guests_out_of_range": {http.StatusBadRequest, []string{"guests"}},
	"unknown_room":        {http.StatusBadRequest, []string{"room"}},
	"bad_email":           {http.StatusBadRequest, []string{"email"}},
	"blank_name":          {http.StatusBadRequest, []string{"name"}},
	"control_character":   {http.StatusBadRequest, []string{"name"}},
}

func TestAttacksAreRejected(t *testing.T) {
	attacks := before.Attacks()
	if len(attacks) != len(want) {
		t.Fatalf("%d attacks but %d expectations", len(attacks), len(want))
	}
	for _, a := range attacks {
		t.Run(a.Name, func(t *testing.T) {
			// before 是對照組：同一份輸入被接受，而且到了 save。
			var reached int
			oracle := before.NewHandler(func(before.Request) { reached++ })
			rec := httptest.NewRecorder()
			oracle.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/bookings", strings.NewReader(a.Body)))
			if rec.Code != http.StatusCreated || reached != 1 {
				t.Fatalf("before: status %d, saved %d; the input should be accepted", rec.Code, reached)
			}

			rec, errs, saved := post(t, strings.NewReader(a.Body))
			w := want[a.Name]
			if rec.Code != w.status {
				t.Errorf("status = %d, want %d", rec.Code, w.status)
			}
			if got := fields(errs); !slices.Equal(got, w.fields) {
				t.Errorf("fields = %q, want %q", got, w.fields)
			}
			if len(saved) != 0 {
				t.Errorf("save was called with %d bookings", len(saved))
			}
			for _, internal := range []string{"json:", "struct", "Request", "booking.", "unmarshal"} {
				if strings.Contains(rec.Body.String(), internal) {
					t.Errorf("response leaks %q: %s", internal, rec.Body)
				}
			}
		})
	}
}

// ANCHOR_END: attacks

// ANCHOR: all

func TestReportsEveryFieldAtOnce(t *testing.T) {
	rec, errs, saved := post(t, strings.NewReader(`{"name":"","email":"nope","room":"","guests":0}`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if got, w := fields(errs), []string{"name", "email", "room", "guests"}; !slices.Equal(got, w) {
		t.Errorf("fields = %q, want %q", got, w)
	}
	if len(saved) != 0 {
		t.Errorf("save was called")
	}
}

// ANCHOR_END: all

func TestWrongTypeNamesTheFieldOnly(t *testing.T) {
	rec, errs, _ := post(t, strings.NewReader(`{"name":"Andy","email":"andy@example.com","room":"a1","guests":"4"}`))
	if rec.Code != http.StatusBadRequest || !slices.Equal(fields(errs), []string{"guests"}) {
		t.Fatalf("status %d, fields %q", rec.Code, fields(errs))
	}
	if strings.Contains(rec.Body.String(), "struct") {
		t.Errorf("response leaks Go type details: %s", rec.Body)
	}
}

func TestMalformedBody(t *testing.T) {
	for _, body := range []string{``, `{`, `[]`, `"x"`} {
		rec, errs, _ := post(t, strings.NewReader(body))
		if rec.Code != http.StatusBadRequest || !slices.Equal(fields(errs), []string{""}) {
			t.Errorf("%q: status %d, fields %q", body, rec.Code, fields(errs))
		}
	}
}

// ANCHOR: bigbody

func TestOversizedBodyIsNotReadToTheEnd(t *testing.T) {
	var big string
	for _, a := range before.Attacks() {
		if a.Name == "oversized_body" {
			big = a.Body
		}
	}
	r := &countingReader{r: strings.NewReader(big)}
	rec, _, _ := post(t, r)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", rec.Code)
	}
	if r.n > 2*booking.MaxBodyBytes {
		t.Errorf("read %d bytes of a %d byte body; limit is %d", r.n, len(big), booking.MaxBodyBytes)
	}
}

// countingReader 記下下游讀走了多少位元組。
type countingReader struct {
	r io.Reader
	n int
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += n
	return n, err
}

// ANCHOR_END: bigbody

func TestValidRequestIsNormalizedThenSaved(t *testing.T) {
	rec, _, saved := post(t, strings.NewReader(`{"name":"  Andy  ","email":" andy@example.com ","room":"a1","guests":2}`))
	if rec.Code != http.StatusCreated || len(saved) != 1 {
		t.Fatalf("status %d, saved %d", rec.Code, len(saved))
	}
	b := saved[0]
	if b.Name() != "Andy" || b.Email() != "andy@example.com" || b.Room() != "a1" || b.Guests() != 2 {
		t.Errorf("saved %q %q %q %d", b.Name(), b.Email(), b.Room(), b.Guests())
	}
}

func TestValidateBoundaries(t *testing.T) {
	tests := []struct {
		name string
		req  booking.Request
		ok   bool
	}{
		{"guests 1", booking.Request{Name: "A", Email: "a@example.com", Room: "a1", Guests: 1}, true},
		{"guests 20", booking.Request{Name: "A", Email: "a@example.com", Room: "b1", Guests: 20}, true},
		{"guests 0", booking.Request{Name: "A", Email: "a@example.com", Room: "a1", Guests: 0}, false},
		{"guests 21", booking.Request{Name: "A", Email: "a@example.com", Room: "a1", Guests: 21}, false},
		{"guests -3", booking.Request{Name: "A", Email: "a@example.com", Room: "a1", Guests: -3}, false},
		{"50 runes", booking.Request{Name: strings.Repeat("好", 50), Email: "a@example.com", Room: "a1", Guests: 1}, true},
		{"51 runes", booking.Request{Name: strings.Repeat("好", 51), Email: "a@example.com", Room: "a1", Guests: 1}, false},
		{"display name in email", booking.Request{Name: "A", Email: "Andy <a@example.com>", Room: "a1", Guests: 1}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.req.Validate()
			if (err == nil) != tt.ok {
				t.Errorf("Validate() error = %v, want ok=%v", err, tt.ok)
			}
		})
	}
}
