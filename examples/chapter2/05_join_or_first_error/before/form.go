// Package form 驗證註冊表單。這是「遇到第一個錯誤就回傳」的版本。
package form

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Form 是使用者送出的註冊表單。
type Form struct {
	Name  string
	Email string
	Age   string
}

// ANCHOR: validate

// Validate 檢查表單，回傳遇到的第一個錯誤。
func Validate(f Form) error {
	if f.Name == "" {
		return errors.New("name is required")
	}
	if !strings.Contains(f.Email, "@") {
		return errors.New("email must contain @")
	}
	age, err := strconv.Atoi(f.Age)
	if err != nil {
		return fmt.Errorf("age must be a number: %w", err)
	}
	if age < 0 || age > 150 {
		return errors.New("age must be between 0 and 150")
	}
	return nil
}

// ANCHOR_END: validate
