package preview_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	preview "github.com/holmeslin/learning-go-to-senior/examples/chapter11/07_ssrf_outbound_requests/after"
)

// ANCHOR: helpers

// serve 啟動一個 loopback 的假伺服器，並數它收到幾個請求。
func serve(t *testing.T, h http.HandlerFunc) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		h(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

// only 是注入的允許清單：只放行這幾台假伺服器的位址。
// httptest 的伺服器都在 loopback，預設規則一定會擋下，所以測試要自己開洞。
func only(servers ...*httptest.Server) func(netip.AddrPort) bool {
	return func(ap netip.AddrPort) bool {
		for _, s := range servers {
			if ap == netip.MustParseAddrPort(s.Listener.Addr().String()) {
				return true
			}
		}
		return false
	}
}

// blocked 取出被擋下的位址；err 不是 *BlockedError 時第二個值是 false。
func blocked(err error) (netip.AddrPort, bool) {
	be, ok := errors.AsType[*preview.BlockedError](err)
	if !ok {
		return netip.AddrPort{}, false
	}
	return be.Addr, true
}

// ANCHOR_END: helpers

// ANCHOR: blocked

func TestLoopbackBlockedByDefault(t *testing.T) {
	srv, hits := serve(t, func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("internal")) })

	_, err := (&preview.Fetcher{}).Fetch(context.Background(), srv.URL)

	if ap, ok := blocked(err); !ok || ap.Addr() != netip.MustParseAddr("127.0.0.1") {
		t.Fatalf("err = %v, want a *BlockedError for 127.0.0.1", err)
	}
	if n := hits.Load(); n != 0 {
		t.Errorf("server saw %d requests, want 0: the connection must be refused before any request is sent", n)
	}
}

// Allow 只增不減：放行它指定的位址，公開位址不必列進去，其他內部位址仍然擋下。
func TestAllowAddsToPublic(t *testing.T) {
	f := &preview.Fetcher{Allow: func(ap netip.AddrPort) bool { return ap.Addr() == netip.MustParseAddr("10.0.0.5") }}

	cases := []struct {
		addr string
		want bool
	}{
		{"10.0.0.5:80", true},
		{"10.0.0.6:80", false},
		{"127.0.0.1:80", false},
		{"8.8.8.8:443", true},
		{"[2606:4700:4700::1111]:443", true},
	}
	for _, c := range cases {
		if got := preview.Allowed(f, netip.MustParseAddrPort(c.addr)); got != c.want {
			t.Errorf("Allowed(%s) = %v, want %v", c.addr, got, c.want)
		}
	}
	if !preview.Allowed(&preview.Fetcher{}, netip.MustParseAddrPort("8.8.8.8:443")) {
		t.Error("with a nil Allow, a public address must be allowed")
	}
}

func TestAllowedAddressConnects(t *testing.T) {
	srv, _ := serve(t, func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("hello")) })

	body, err := (&preview.Fetcher{Allow: only(srv)}).Fetch(context.Background(), srv.URL)
	if err != nil || string(body) != "hello" {
		t.Fatalf("Fetch = %q, %v; want hello, nil", body, err)
	}
}

// ANCHOR_END: blocked

// ANCHOR: redirect

func TestRedirectToPrivateBlocked(t *testing.T) {
	internal, internalHits := serve(t, func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("internal")) })
	public, _ := serve(t, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, internal.URL, http.StatusFound)
	})

	f := &preview.Fetcher{Allow: only(public), MaxRedirects: 3}
	_, err := f.Fetch(context.Background(), public.URL)

	want := netip.MustParseAddrPort(internal.Listener.Addr().String())
	if ap, ok := blocked(err); !ok || ap != want {
		t.Fatalf("err = %v, want a *BlockedError for the redirect target %v", err, want)
	}
	if n := internalHits.Load(); n != 0 {
		t.Errorf("internal server saw %d requests, want 0", n)
	}
}

func TestRedirectLimit(t *testing.T) {
	var srv *httptest.Server
	srv, hits := serve(t, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, srv.URL, http.StatusFound) // 永遠轉回自己
	})

	_, err := (&preview.Fetcher{Allow: only(srv), MaxRedirects: 2}).Fetch(context.Background(), srv.URL)

	if err == nil || !strings.Contains(err.Error(), "too many redirects") {
		t.Fatalf("err = %v, want too many redirects", err)
	}
	if n := hits.Load(); n != 3 { // 第一次加上 2 次轉址
		t.Errorf("server saw %d requests, want 3", n)
	}
}

// ANCHOR_END: redirect

// ANCHOR: limits

func TestTimeout(t *testing.T) {
	srv, _ := serve(t, func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() })

	f := &preview.Fetcher{Allow: only(srv), Timeout: 20 * time.Millisecond}
	_, err := f.Fetch(context.Background(), srv.URL)

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want a deadline exceeded error", err)
	}
}

func TestBodyLimit(t *testing.T) {
	srv, _ := serve(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(strings.Repeat("x", 11)))
	})

	_, err := (&preview.Fetcher{Allow: only(srv), MaxBody: 10}).Fetch(context.Background(), srv.URL)
	if err == nil || !strings.Contains(err.Error(), "exceeds 10 bytes") {
		t.Fatalf("11 bytes with MaxBody 10: err = %v, want exceeds 10 bytes", err)
	}

	body, err := (&preview.Fetcher{Allow: only(srv), MaxBody: 11}).Fetch(context.Background(), srv.URL)
	if err != nil || len(body) != 11 {
		t.Fatalf("11 bytes with MaxBody 11: len = %d, err = %v; want 11, nil", len(body), err)
	}
}

// ANCHOR_END: limits

// ANCHOR: ispublic

func TestIsPublic(t *testing.T) {
	cases := map[string]bool{
		"8.8.8.8":              true,
		"2606:4700:4700::1111": true,
		"127.0.0.1":            false,
		"::1":                  false,
		"0.0.0.0":              false,
		"::":                   false,
		"10.0.0.1":             false,
		"172.16.0.1":           false,
		"192.168.1.1":          false,
		"fd00::1":              false, // 唯一本地位址
		"169.254.169.254":      false, // link-local，雲端的 metadata 端點常放在這一段
		"fe80::1":              false,
		"100.64.0.1":           false, // CGNAT
		"224.0.0.1":            false,
		"255.255.255.255":      false,
		"::ffff:127.0.0.1":     false, // IPv4-mapped
		"::ffff:100.64.0.1":    false,
		"64:ff9b::7f00:1":      false, // NAT64 內嵌 127.0.0.1
		"192.0.0.1":            false, // IETF 保留
		"198.18.0.1":           false, // 效能測試用（198.18.0.0/15 的頭）
		"198.19.255.255":       false, // 同一段的尾
		"240.0.0.1":            false, // 保留
		"2002:808:808::1":      false, // 6to4 內嵌 8.8.8.8
		"2001::1":              false, // Teredo
		"100.128.0.1":          true,  // 特殊範圍的邊界外一格：不能誤擋
		"192.0.1.1":            true,
		"198.20.0.1":           true,
	}
	for s, want := range cases {
		if got := preview.IsPublic(netip.MustParseAddr(s)); got != want {
			t.Errorf("IsPublic(%s) = %v, want %v", s, got, want)
		}
	}
}

// ANCHOR_END: ispublic
