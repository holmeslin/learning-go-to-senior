// Package commands 是一個命令註冊表：名稱對應到處理函式。
package commands

import (
	"errors"
	"fmt"
	"maps"
	"slices"
)

// ErrDuplicate 表示同一個名稱註冊了兩次。
var ErrDuplicate = errors.New("already registered")

// ErrUnknown 表示執行了沒有註冊過的名稱。
var ErrUnknown = errors.New("unknown command")

// Handler 處理一個命令，回傳要顯示的結果。
type Handler func(args []string) string

// ANCHOR: registry

// Registry 保存名稱與處理函式的對應。零值不能用，請用 New 建立。
type Registry struct {
	handlers map[string]Handler
}

// New 建立一個空的 Registry。每個呼叫端各拿一份，不共用狀態。
func New() *Registry {
	return &Registry{handlers: map[string]Handler{}}
}

// ANCHOR_END: registry

// Register 註冊 name。name 已存在時回傳包裝 ErrDuplicate 的錯誤。
func (r *Registry) Register(name string, h Handler) error {
	if _, ok := r.handlers[name]; ok {
		return fmt.Errorf("register %q: %w", name, ErrDuplicate)
	}
	r.handlers[name] = h
	return nil
}

// Run 執行 name 對應的處理函式。name 沒有註冊時回傳包裝 ErrUnknown 的錯誤。
func (r *Registry) Run(name string, args ...string) (string, error) {
	h, ok := r.handlers[name]
	if !ok {
		return "", fmt.Errorf("run %q: %w", name, ErrUnknown)
	}
	return h(args), nil
}

// Names 回傳已註冊的名稱，依字母順序排列。
func (r *Registry) Names() []string {
	return slices.Sorted(maps.Keys(r.handlers))
}
