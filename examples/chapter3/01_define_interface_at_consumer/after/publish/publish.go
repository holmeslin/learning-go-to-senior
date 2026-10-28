// Package publish 把草稿發布成正式內容。
package publish

import (
	"bytes"
	"fmt"
)

// ANCHOR: blobs

// Blobs 是 publish 對儲存的需求：讀草稿、寫成品。
// 介面由使用它的 publish 宣告，只列出 Publish 真的會呼叫的方法。
type Blobs interface {
	Get(key string) ([]byte, error)
	Put(key string, data []byte) error
}

// ANCHOR_END: blobs

// ANCHOR: publish

// Publish 讀出 drafts/<id>，去掉前後空白，寫到 published/<id>。
func Publish(b Blobs, id string) error {
	draft, err := b.Get("drafts/" + id)
	if err != nil {
		return fmt.Errorf("publish %s: read draft: %w", id, err)
	}
	if err = b.Put("published/"+id, bytes.TrimSpace(draft)); err != nil {
		return fmt.Errorf("publish %s: write: %w", id, err)
	}
	return nil
}

// ANCHOR_END: publish
