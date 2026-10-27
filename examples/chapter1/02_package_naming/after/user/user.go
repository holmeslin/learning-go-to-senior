// Package user 管理使用者的註冊。
package user

import (
	"errors"
	"slices"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter1/02_package_naming/after/slug"
)

// ErrReserved 表示網址代稱是保留字，不能註冊。
var ErrReserved = errors.New("user: reserved name")

// User 是一個已註冊的使用者。
type User struct {
	Name string
	Slug string
}

// ANCHOR: service

// Service 負責使用者的註冊。請用 NewService 建立。
type Service struct {
	reserved []string
}

// NewService 建立一個 Service，預設保留 admin 與 root。
func NewService() *Service {
	return &Service{reserved: []string{"admin", "root"}}
}

// ANCHOR_END: service

// Register 以 name 建立使用者；網址代稱是保留字時回傳 ErrReserved。
func (s *Service) Register(name string) (User, error) {
	sl := slug.Make(name)
	if slices.Contains(s.reserved, sl) {
		return User{}, ErrReserved
	}
	return User{Name: name, Slug: sl}, nil
}
