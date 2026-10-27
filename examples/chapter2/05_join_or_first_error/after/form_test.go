package form_test

import (
	"errors"
	"slices"
	"strconv"
	"testing"

	form "github.com/holmeslin/learning-go-to-senior/examples/chapter2/05_join_or_first_error/after"
)

// fields 取出合併錯誤裡每個 *FieldError 的欄位名稱。
func fields(t *testing.T, err error) []string {
	t.Helper()
	joined, ok := err.(interface{ Unwrap() []error })
	if !ok {
		t.Fatalf("error %v does not wrap a list of errors", err)
	}
	var got []string
	for _, e := range joined.Unwrap() {
		fe, ok := errors.AsType[*form.FieldError](e)
		if !ok {
			t.Fatalf("%v is not a *form.FieldError", e)
		}
		got = append(got, fe.Field)
	}
	return got
}

func TestValidateReportsEveryField(t *testing.T) {
	got := fields(t, form.Validate(form.Form{Age: "200"}))
	if want := []string{"name", "email", "age"}; !slices.Equal(got, want) {
		t.Fatalf("fields = %v; want %v", got, want)
	}
}

func TestValidateErrorsAreStillFindable(t *testing.T) {
	err := form.Validate(form.Form{Email: "andy@example.com", Age: "abc"})
	if !errors.Is(err, form.ErrRequired) {
		t.Errorf("errors.Is(err, ErrRequired) = false; err = %v", err)
	}
	if !errors.Is(err, strconv.ErrSyntax) {
		t.Errorf("errors.Is(err, strconv.ErrSyntax) = false; err = %v", err)
	}
	if fe, ok := errors.AsType[*form.FieldError](err); !ok || fe.Field != "name" {
		t.Errorf("errors.AsType = %v, %v; want the name field first", fe, ok)
	}
}

// age 的範圍檢查要用解析出來的數字，解析失敗就只回報解析的錯誤。
func TestValidateAgeStopsAtFirstError(t *testing.T) {
	err := form.Validate(form.Form{Name: "Andy", Email: "andy@example.com", Age: "abc"})
	if got := fields(t, err); !slices.Equal(got, []string{"age"}) {
		t.Fatalf("fields = %v; want [age]", got)
	}
	if !errors.Is(err, strconv.ErrSyntax) {
		t.Errorf("errors.Is(err, strconv.ErrSyntax) = false; err = %v", err)
	}
}

func TestValidateOK(t *testing.T) {
	if err := form.Validate(form.Form{Name: "Andy", Email: "andy@example.com", Age: "20"}); err != nil {
		t.Errorf("Validate = %v; want nil", err)
	}
}
