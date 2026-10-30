// Package authkit 示範產生 token、比對 token 與設定 TLS 的寫法：
// crypto/rand、常數時間比對、client 信任指定的 CA、伺服器要求最低版本。
package authkit

import (
	"crypto/rand"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net/http"
	"time"
)

// ANCHOR: token

// NewToken 回傳可以當重設碼或 session token 的字串：至少 128 位元的亂數，
// 用 RFC 4648 的 base32 字元（A-Z、2-7）編碼，不含填充字元。
func NewToken() string { return rand.Text() }

// EqualToken 以常數時間比對使用者送來的 token 與資料庫裡的那一個。
// 長度不同時 ConstantTimeCompare 會直接回 0，token 長度本來就不是秘密。
func EqualToken(got, want string) bool {
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}

// ANCHOR_END: token

// ANCHOR: tls

// NewClient 回傳只信任 caPEM 裡那些 CA 的 http.Client，用來呼叫簽發自私有 CA 的內部服務。
// 驗證保持開啟，其他 CA 簽的憑證一律拒絕；caPEM 裡沒有任何憑證時回傳錯誤。
func NewClient(caPEM []byte) (*http.Client, error) {
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return nil, errors.New("no certificate found in ca pem")
	}
	return &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12},
		},
	}, nil
}

// ServerConfig 回傳內部服務用的伺服器設定：最低版本是 TLS 1.3，
// 因為呼叫端都是自己控制的 client。對外、client 不受控的服務請保留預設的最低版本。
func ServerConfig(cert tls.Certificate) *tls.Config {
	return &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS13}
}

// ANCHOR_END: tls
