package signup_test

import (
	"context"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter4/07_fake_or_real_database/before/signup"
)

// ANCHOR: fake

// fakeStore 用 map 假裝資料表：來者不拒。
type fakeStore struct {
	users map[int64]signup.User
}

func newFakeStore() *fakeStore {
	return &fakeStore{users: map[int64]signup.User{}}
}

func (f *fakeStore) Add(_ context.Context, email string) (signup.User, error) {
	u := signup.User{ID: int64(len(f.users) + 1), Email: email}
	f.users[u.ID] = u
	return u, nil
}

// ANCHOR_END: fake
