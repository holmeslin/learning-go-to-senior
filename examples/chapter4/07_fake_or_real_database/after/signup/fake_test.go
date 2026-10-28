package signup_test

import (
	"context"
	"strings"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter4/07_fake_or_real_database/after/signup"
)

// ANCHOR: fake

// fakeStore 用 map 假裝資料表，並照 signup.Store 的契約行事：
// 信箱不分大小寫，重複就回 ErrEmailTaken。
type fakeStore struct {
	users map[string]signup.User // key 是小寫的信箱
}

func newFakeStore() *fakeStore {
	return &fakeStore{users: map[string]signup.User{}}
}

func (f *fakeStore) Add(_ context.Context, email string) (signup.User, error) {
	key := strings.ToLower(email)
	if _, ok := f.users[key]; ok {
		return signup.User{}, signup.ErrEmailTaken
	}
	u := signup.User{ID: int64(len(f.users) + 1), Email: email}
	f.users[key] = u
	return u, nil
}

// ANCHOR_END: fake
