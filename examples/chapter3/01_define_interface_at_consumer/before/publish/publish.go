// Package publish 把草稿發布成正式內容。
package publish

import (
	"bytes"
	"fmt"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter3/01_define_interface_at_consumer/before/blob"
)

// ANCHOR: publish

// Publish 讀出 drafts/<id>，去掉前後空白，寫到 published/<id>。
func Publish(s blob.Store, id string) error {
	draft, err := s.Get("drafts/" + id)
	if err != nil {
		return fmt.Errorf("publish %s: read draft: %w", id, err)
	}
	if err = s.Put("published/"+id, bytes.TrimSpace(draft)); err != nil {
		return fmt.Errorf("publish %s: write: %w", id, err)
	}
	return nil
}

// ANCHOR_END: publish
