// Package app 是呼叫端：註冊使用者，並組出歡迎訊息。
package app

import (
	"fmt"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter1/02_package_naming/before/user"
	"github.com/holmeslin/learning-go-to-senior/examples/chapter1/02_package_naming/before/util"
)

// ANCHOR: welcome
func Welcome(name, bio string) (string, error) {
	svc := user.NewUserService()
	u, err := svc.Register(name)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("/u/%s %s", u.Slug, util.Truncate(bio, 8)), nil
}

// ANCHOR_END: welcome
