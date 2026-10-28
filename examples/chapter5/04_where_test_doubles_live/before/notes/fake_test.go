package notes

import "github.com/holmeslin/learning-go-to-senior/examples/chapter5/04_where_test_doubles_live/before/kv"

// fakeStore 是 notes 自己抄的一份 kv fake。作者撞到 kv 的新規則之後，
// 在自己這份補上了檢查；其他套件的那一份沒人通知。
type fakeStore struct {
	values map[string]string
}

func newFakeStore() *fakeStore {
	return &fakeStore{values: map[string]string{}}
}

func (f *fakeStore) Put(key, value string) error {
	if !kv.ValidKey(key) {
		return kv.ErrInvalidKey
	}
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
