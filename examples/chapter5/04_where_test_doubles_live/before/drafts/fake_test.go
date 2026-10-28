package drafts

import "github.com/holmeslin/learning-go-to-senior/examples/chapter5/04_where_test_doubles_live/before/kv"

// ANCHOR: fake

// fakeStore 是 drafts 自己抄的一份 kv fake，寫在 kv 加上「key 不可含斜線」之前。
type fakeStore struct {
	values map[string]string
}

func newFakeStore() *fakeStore {
	return &fakeStore{values: map[string]string{}}
}

func (f *fakeStore) Put(key, value string) error {
	f.values[key] = value
	return nil
}

func (f *fakeStore) Get(key string) (string, error) {
	v, ok := f.values[key]
	if !ok {
		return "", kv.ErrNotFound
	}
	return v, nil
}

// ANCHOR_END: fake
