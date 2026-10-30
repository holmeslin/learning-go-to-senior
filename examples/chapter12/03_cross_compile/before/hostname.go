package main

/*
#include <unistd.h>
*/
import "C"

// ANCHOR: hostname

// hostname 呼叫 C 的 gethostname 取得主機名稱。
func hostname() string {
	var buf [256]C.char
	if C.gethostname(&buf[0], C.size_t(len(buf))) != 0 {
		return "unknown"
	}
	return C.GoString(&buf[0])
}

// ANCHOR_END: hostname
