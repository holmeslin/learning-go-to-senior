// Package commands 是一個命令註冊表：名稱對應到處理函式，狀態放在套件層級。
//
// 這個套件的測試刻意帶有順序相依：預設的 go test 會通過，
// 但下面兩個指令會讓它穩定失敗（用 go test -v 可以看到每個測試的結果）：
//
//	go test -shuffle=2 .   # TestRun 排在 TestRegister 前面，找不到 echo
//	go test -count=2 .     # 第二輪 TestRegister 發現 echo 已經註冊過
//
// -shuffle=1 的順序剛好和預設相同，所以會通過；單獨執行 go test -run TestRun . 也會失敗。
package commands

import (
	"errors"
	"fmt"
)

// ErrDuplicate 表示同一個名稱註冊了兩次。
var ErrDuplicate = errors.New("already registered")

// ErrUnknown 表示執行了沒有註冊過的名稱。
var ErrUnknown = errors.New("unknown command")

// Handler 處理一個命令，回傳要顯示的結果。
type Handler func(args []string) string

// ANCHOR: registry

var registry = map[string]Handler{}

// Register 註冊 name。name 已存在時回傳包裝 ErrDuplicate 的錯誤。
func Register(name string, h Handler) error {
	if _, ok := registry[name]; ok {
		return fmt.Errorf("register %q: %w", name, ErrDuplicate)
	}
	registry[name] = h
	return nil
}

// ANCHOR_END: registry

// Run 執行 name 對應的處理函式。name 沒有註冊時回傳包裝 ErrUnknown 的錯誤。
func Run(name string, args ...string) (string, error) {
	h, ok := registry[name]
	if !ok {
		return "", fmt.Errorf("run %q: %w", name, ErrUnknown)
	}
	return h(args), nil
}
