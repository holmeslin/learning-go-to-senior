//go:build unix

package probe

import "syscall"

// ANCHOR: unix

// FileLimit 回傳行程能同時開啟的檔案數上限（RLIMIT_NOFILE 的軟限制）。
func FileLimit() (uint64, error) {
	var lim syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &lim); err != nil {
		return 0, err
	}
	return uint64(lim.Cur), nil
}

// ANCHOR_END: unix
