// Package signup 是常見寫法：用例測試與資料層都靠 map 做的 fake，真的資料庫從來沒被測過。
package signup

import (
	"context"
	"errors"
)

// ErrEmailTaken 表示這個信箱已經註冊過。
var ErrEmailTaken = errors.New("email already registered")

// User 是一個已註冊的使用者。
type User struct {
	ID    int64  // 資料庫配給的主鍵
	Email string // 註冊時給的信箱，保留原本的大小寫
}

// ANCHOR: service

// Store 是 Service 需要的儲存能力，宣告在使用它的這一端。
type Store interface {
	// Add 新增一個信箱。信箱已經註冊過時回傳 ErrEmailTaken。
	Add(ctx context.Context, email string) (User, error)
}

// Service 處理註冊相關的用例。
type Service struct {
	store Store
}

// NewService 回傳使用 store 的 Service。
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Invite 註冊 emails 裡的每個信箱，回傳新增的筆數。
// 已經註冊過的信箱會略過並一併回傳；其他錯誤則中止，並回傳到目前為止的結果。
func (s *Service) Invite(ctx context.Context, emails []string) (added int, skipped []string, err error) {
	for _, email := range emails {
		_, err = s.store.Add(ctx, email)
		switch {
		case err == nil:
			added++
		case errors.Is(err, ErrEmailTaken):
			skipped = append(skipped, email)
		default:
			return added, skipped, err
		}
	}
	return added, skipped, nil
}

// ANCHOR_END: service
