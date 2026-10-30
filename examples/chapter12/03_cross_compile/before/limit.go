package main

import "syscall"

// ANCHOR: limit

// fileLimit 回傳行程能同時開啟的檔案數上限。檔案沒有 build tag，
// 每個平台都會編譯，但 syscall.Getrlimit 不是每個平台都有。
func fileLimit() (uint64, error) {
	var lim syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &lim); err != nil {
		return 0, err
	}
	return uint64(lim.Cur), nil
}

// ANCHOR_END: limit
