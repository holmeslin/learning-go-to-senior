// Package preview 是連結預覽功能：伺服器替使用者抓取網址。
// 防線放在連線建立的那一刻，檢查實際要連的 IP，並限制轉址、逾時與回應大小。
package preview

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"syscall"
	"time"
)

// ANCHOR: policy

// BlockedError 表示連線的目的位址沒有通過檢查。
type BlockedError struct {
	Addr netip.AddrPort // 被擋下的、實際要連的位址
}

// Error 實作 error。訊息帶著位址，是給日誌看的；回給使用者的訊息要另外換成一致的說法。
func (e *BlockedError) Error() string { return "address " + e.Addr.String() + " is not allowed" }

// 公開位址以外、netip 的方法沒有涵蓋的特殊範圍。
var special = []netip.Prefix{
	netip.MustParsePrefix("100.64.0.0/10"), // CGNAT
	netip.MustParsePrefix("192.0.0.0/24"),  // IETF 保留
	netip.MustParsePrefix("198.18.0.0/15"), // 效能測試用
	netip.MustParsePrefix("240.0.0.0/4"),   // 保留
	netip.MustParsePrefix("64:ff9b::/96"),  // NAT64，內嵌的 IPv4 可能是內網位址
	netip.MustParsePrefix("2002::/16"),     // 6to4
	netip.MustParsePrefix("2001::/32"),     // Teredo
}

// IsPublic 回報 a 是不是公開的單播位址。loopback、私有、link-local、
// 未指定、多播，以及上面列出的特殊範圍都不是；IPv4-mapped 的 IPv6 位址先還原成 IPv4 再判斷。
func IsPublic(a netip.Addr) bool {
	a = a.Unmap()
	if !a.IsGlobalUnicast() || a.IsPrivate() {
		return false
	}
	for _, p := range special {
		if p.Contains(a) {
			return false
		}
	}
	return true
}

// ANCHOR_END: policy

// ANCHOR: fetcher

// Fetcher 替使用者抓取網址。零值可以直接用：只連公開位址、最多 5 秒、
// 不跟隨轉址、回應最多 1 MiB。
type Fetcher struct {
	// Allow 決定實際要連的位址能不能連；nil 時只放行 IsPublic 的位址。
	// 這個欄位也是測試的接縫：httptest 的伺服器在 loopback，預設規則一定擋下，
	// 測試要用它只放行指定的假伺服器；正式環境可以用它放行少數已知的內部位址。
	Allow func(netip.AddrPort) bool

	// Resolver 解析主機名稱；nil 時用 net.DefaultResolver。
	// 測試注入會把名稱解析到 loopback 的 *net.Resolver，不用真的 DNS。
	Resolver *net.Resolver

	Timeout      time.Duration // 整個請求（含轉址、讀取回應）的時限；0 代表 5 秒
	MaxRedirects int           // 最多跟隨幾次轉址；0 代表不跟隨
	MaxBody      int64         // 回應內容的位元組上限；0 代表 1 MiB
}

// control 在連線建立前被呼叫，address 是已經解析好、實際要連的 "ip:port"。
// 每次連線、每一次轉址、每個解析出來的位址都會經過這裡。
func (f *Fetcher) control(_, address string, _ syscall.RawConn) error {
	ap, err := netip.ParseAddrPort(address)
	if err != nil {
		return err
	}
	ok := IsPublic(ap.Addr())
	if f.Allow != nil {
		ok = f.Allow(ap)
	}
	if !ok {
		return &BlockedError{Addr: ap}
	}
	return nil
}

func (f *Fetcher) client() *http.Client {
	dialer := &net.Dialer{Control: f.control, Resolver: f.Resolver}
	return &http.Client{
		Timeout: cmp.Or(f.Timeout, 5*time.Second),
		// 不設 Proxy：經過 proxy 時 Control 看到的是 proxy 的位址，不是目的地。
		Transport: &http.Transport{DialContext: dialer.DialContext, DisableKeepAlives: true},
		CheckRedirect: func(_ *http.Request, via []*http.Request) error {
			if len(via) > f.MaxRedirects {
				return fmt.Errorf("too many redirects (max %d)", f.MaxRedirects)
			}
			return nil
		},
	}
}

// ANCHOR_END: fetcher

// ANCHOR: fetch

// Fetch 抓取 rawURL 的內容。實際連線的位址沒通過檢查時，回傳的錯誤可以用
// errors.AsType[*BlockedError] 取出；回應超過 MaxBody 也是錯誤，不會截斷後回傳。
func (f *Fetcher) Fetch(ctx context.Context, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := f.client().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	limit := cmp.Or(f.MaxBody, 1<<20)
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > limit {
		return nil, fmt.Errorf("response body exceeds %d bytes", limit)
	}
	return body, nil
}

// ANCHOR_END: fetch
