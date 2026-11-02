package signup_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	signup "github.com/holmeslin/learning-go-to-senior/examples/chapter13/01_design_before_details/after"
)

// do 對 handler 送出一個請求，回傳狀態碼與回應本文。
func do(t *testing.T, h http.Handler, method, target, body string) (int, string) {
	t.Helper()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code, rec.Body.String()
}

func TestHandlerRegisterAndCancel(t *testing.T) {
	store := signup.NewStore(&fakeNotifier{})
	if _, err := store.Create("go-meetup", 1, 10); err != nil {
		t.Fatal(err)
	}
	h := signup.NewHandler(store)

	if code, _ := do(t, h, "POST", "/events/go-meetup/registrations", `{"who":"u1"}`); code != http.StatusCreated {
		t.Fatalf("first register = %d, want 201", code)
	}

	code, body := do(t, h, "GET", "/events/go-meetup", "")
	if code != http.StatusOK {
		t.Fatalf("GET = %d, want 200", code)
	}
	var view struct {
		Capacity  int      `json:"capacity"`
		Attendees []string `json:"attendees"`
	}
	if err := json.Unmarshal([]byte(body), &view); err != nil {
		t.Fatal(err)
	}
	if view.Capacity != 1 || !slices.Equal(view.Attendees, []string{"u1"}) {
		t.Errorf("GET body = %+v, want capacity 1 and attendees [u1]", view)
	}

	if code, _ := do(t, h, "DELETE", "/events/go-meetup/registrations/u1", ""); code != http.StatusNoContent {
		t.Errorf("cancel = %d, want 204", code)
	}
	if code, _ := do(t, h, "DELETE", "/events/go-meetup/registrations/u1", ""); code != http.StatusNotFound {
		t.Errorf("second cancel = %d, want 404", code)
	}
	if code, _ := do(t, h, "GET", "/events/missing", ""); code != http.StatusNotFound {
		t.Errorf("GET missing event = %d, want 404", code)
	}
}
