// Package relay 是刻意示範的壞寫法：每個地方用到設定才去讀環境變數，
// 讀不到、格式錯都悄悄換成別的值。不要照用。
package relay

import (
	"net/http"
	"os"
	"strconv"
	"time"
)

// ANCHOR: client

// Upstream 回傳上游的網址；沒設就用本機的 8080。
func Upstream() string {
	if u := os.Getenv("RELAY_UPSTREAM"); u != "" {
		return u
	}
	return "http://localhost:8080"
}

// NewClient 建立呼叫上游的 client，逾時讀 RELAY_TIMEOUT_SEC（秒）。
func NewClient() *http.Client {
	n, _ := strconv.Atoi(os.Getenv("RELAY_TIMEOUT_SEC"))
	return &http.Client{Timeout: time.Duration(n) * time.Second}
}

// MaxRetries 回傳失敗後最多重試幾次，讀 RELAY_MAX_RETRIES。
func MaxRetries() int {
	n, _ := strconv.Atoi(os.Getenv("RELAY_MAX_RETRIES"))
	return n
}

// ANCHOR_END: client
