// Package user 管理使用者的註冊。
package user

import (
	"errors"
	"slices"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter1/02_package_naming/after/slug"
)

var ErrReserved = errors.New("user: reserved name")

type User struct {
	Name string
	Slug string
}

// ANCHOR: service
type Service struct {
	reserved []string
}

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
