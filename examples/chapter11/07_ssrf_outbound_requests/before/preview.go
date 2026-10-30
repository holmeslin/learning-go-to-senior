// Package preview 是連結預覽功能：伺服器替使用者抓取網址。
//
// 這個套件是刻意示範的漏洞（SSRF），不要照用：它只檢查網址的字串，
// 不檢查實際連到哪裡，也沒有逾時與回應大小的上限。較好的寫法在 after 套件。
package preview

import (
	"errors"
	"io"
	"net/http"
	"net/netip"
	"net/url"
)

// ANCHOR: fetch

// Fetch 抓取使用者給的 rawURL。刻意示範的漏洞，不要照用：
// 先檢查網址的字串，通過就直接請求。client 傳 http.DefaultClient，
// 就等於 http.Get(rawURL)；做成參數，是為了讓測試能指定名稱解析到哪裡。
func Fetch(client *http.Client, rawURL string) ([]byte, error) {
	if err := checkURL(rawURL); err != nil {
		return nil, err
	}
	resp, err := client.Get(rawURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// checkURL 只看字串：只收 http 與 https，主機不能是 localhost，
// 也不能是 loopback、私有、link-local 或未指定的 IP 字面值。
func checkURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.New("scheme is not allowed")
	}
	host := u.Hostname()
	if host == "localhost" {
		return errors.New("host is not allowed")
	}
	if a, err := netip.ParseAddr(host); err == nil &&
		(a.IsLoopback() || a.IsPrivate() || a.IsLinkLocalUnicast() || a.IsUnspecified()) {
		return errors.New("host is not allowed")
	}
	return nil
}

// ANCHOR_END: fetch
