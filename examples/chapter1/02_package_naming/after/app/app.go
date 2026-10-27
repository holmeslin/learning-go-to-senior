// Package app 是呼叫端：註冊使用者，並組出歡迎訊息。
package app

import (
	"fmt"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter1/02_package_naming/after/text"
	"github.com/holmeslin/learning-go-to-senior/examples/chapter1/02_package_naming/after/user"
)

// ANCHOR: welcome

// Welcome 註冊 name，回傳「網址 自我介紹摘要」形式的歡迎訊息；註冊失敗時回傳錯誤。
func Welcome(name, bio string) (string, error) {
	svc := user.NewService()
	u, err := svc.Register(name)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("/u/%s %s", u.Slug, text.Truncate(bio, 8)), nil
}

// ANCHOR_END: welcome
