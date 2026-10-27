// Package user 管理使用者的註冊。
package user

import (
	"errors"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter1/02_package_naming/before/util"
)

var ErrReserved = errors.New("user: reserved name")

type User struct {
	Name string
	Slug string
}

// ANCHOR: service
type UserService struct {
	reserved []string
}

func NewUserService() *UserService {
	return &UserService{reserved: []string{"admin", "root"}}
}

// ANCHOR_END: service

func (s *UserService) Register(name string) (User, error) {
	slug := util.Slugify(name)
	if util.Contains(s.reserved, slug) {
		return User{}, ErrReserved
	}
	return User{Name: name, Slug: slug}, nil
}
