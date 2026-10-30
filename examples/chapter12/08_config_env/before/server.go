package relay

import "os"

// ANCHOR: server

// Addr 回傳監聽位址，埠號讀 RELAY_PORT；沒設就用 0。
func Addr() string {
	port := os.Getenv("RELAY_PORT")
	if port == "" {
		port = "0"
	}
	return ":" + port
}

// Debug 回傳是否輸出除錯日誌：RELAY_DEBUG 有設就是開。
func Debug() bool {
	return os.Getenv("RELAY_DEBUG") != ""
}

// ANCHOR_END: server
