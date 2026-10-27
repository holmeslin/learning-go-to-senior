// Package form 驗證註冊表單，一次回報所有有問題的欄位。
package form

import (
	"errors"
	"strconv"
	"strings"
)

// ErrRequired 表示必填欄位是空的。
var ErrRequired = errors.New("required")

// Form 是使用者送出的註冊表單。
type Form struct {
	Name  string
	Email string
	Age   string
}

// ANCHOR: fielderror

// FieldError 說明哪一個欄位有什麼問題。
type FieldError struct {
	Field string
	Err   error
}

// Error 回傳「欄位: 原因」。
func (e *FieldError) Error() string { return e.Field + ": " + e.Err.Error() }

// Unwrap 讓 [errors.Is] 與 [errors.As] 找得到底下的原因。
func (e *FieldError) Unwrap() error { return e.Err }

// ANCHOR_END: fielderror

// ANCHOR: validate

// Validate 檢查表單。沒有問題時回傳 nil；否則回傳 [errors.Join] 合併的錯誤，
// 每個有問題的欄位各是一個 [*FieldError]，順序依欄位在表單裡的順序。
func Validate(f Form) error {
	var errs []error
	if f.Name == "" {
		errs = append(errs, &FieldError{Field: "name", Err: ErrRequired})
	}
	if !strings.Contains(f.Email, "@") {
		errs = append(errs, &FieldError{Field: "email", Err: errors.New("must contain @")})
	}
	if err := checkAge(f.Age); err != nil {
		errs = append(errs, &FieldError{Field: "age", Err: err})
	}
	return errors.Join(errs...)
}

// ANCHOR_END: validate

// ANCHOR: checkage

// checkAge 的範圍檢查要用解析出來的數字，所以解析失敗就直接回傳，不再往下。
func checkAge(s string) error {
	n, err := strconv.Atoi(s)
	if err != nil {
		return err
	}
	if n < 0 || n > 150 {
		return errors.New("must be between 0 and 150")
	}
	return nil
}

// ANCHOR_END: checkage
