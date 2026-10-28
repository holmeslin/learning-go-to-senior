// Package blob 是一個放在記憶體裡的物件儲存，用來代表會碰到 I/O 的儲存服務。
// 它只匯出具體型別，不替使用端宣告介面。
package blob

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// ErrNotFound 表示 key 不存在。
var ErrNotFound = errors.New("not found")

// Bucket 以 key 存放資料。Bucket 不是並行安全的；請用 NewBucket 建立。
type Bucket struct {
	data map[string][]byte
}

// NewBucket 建立一個空的 Bucket。
func NewBucket() *Bucket {
	return &Bucket{data: map[string][]byte{}}
}

// Get 回傳 key 的內容；key 不存在時，錯誤會包住 ErrNotFound。
func (b *Bucket) Get(key string) ([]byte, error) {
	data, ok := b.data[key]
	if !ok {
		return nil, fmt.Errorf("get %q: %w", key, ErrNotFound)
	}
	return data, nil
}

// Put 寫入 key，已存在就覆蓋。
func (b *Bucket) Put(key string, data []byte) error {
	b.data[key] = data
	return nil
}

// Delete 刪除 key；key 不存在時，錯誤會包住 ErrNotFound。
func (b *Bucket) Delete(key string) error {
	if _, ok := b.data[key]; !ok {
		return fmt.Errorf("delete %q: %w", key, ErrNotFound)
	}
	delete(b.data, key)
	return nil
}

// Exists 回報 key 是否存在。
func (b *Bucket) Exists(key string) bool {
	_, ok := b.data[key]
	return ok
}

// List 回傳以 prefix 開頭的 key，依字母順序排列。
func (b *Bucket) List(prefix string) []string {
	var keys []string
	for key := range b.data {
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
	}
	slices.Sort(keys)
	return keys
}

// Copy 把 src 的內容複製到 dst；src 不存在時，錯誤會包住 ErrNotFound。
func (b *Bucket) Copy(src, dst string) error {
	data, err := b.Get(src)
	if err != nil {
		return fmt.Errorf("copy to %q: %w", dst, err)
	}
	return b.Put(dst, slices.Clone(data))
}

// Close 釋放資源。記憶體版沒有東西要釋放。
func (b *Bucket) Close() error {
	return nil
}
