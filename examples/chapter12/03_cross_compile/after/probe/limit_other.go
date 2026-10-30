//go:build !unix

package probe

import "errors"

// ANCHOR: other

// FileLimit 在沒有 RLIMIT_NOFILE 的平台回傳 errors.ErrUnsupported。
func FileLimit() (uint64, error) {
	return 0, errors.ErrUnsupported
}

// ANCHOR_END: other
