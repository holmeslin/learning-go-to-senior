package authkit_test

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"strings"
	"testing"

	authkit "github.com/holmeslin/learning-go-to-senior/examples/chapter11/06_crypto_rand_tls/after"
)

// ANCHOR: token

func TestNewToken(t *testing.T) {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567" // RFC 4648 base32
	tok := authkit.NewToken()
	// rand.Text 保證至少 128 位元，base32 每個字元 5 位元，所以至少 26 個字元。
	if len(tok) < 26 {
		t.Errorf("len = %d, want >= 26", len(tok))
	}
	for _, c := range tok {
		if !strings.ContainsRune(alphabet, c) {
			t.Errorf("unexpected character %q in %q", c, tok)
		}
	}
}

func TestEqualToken(t *testing.T) {
	tests := []struct {
		name      string
		got, want string
		equal     bool
	}{
		{"same", "ABCDEFGH", "ABCDEFGH", true},
		{"one byte differs", "ABCDEFGX", "ABCDEFGH", false},
		{"prefix", "ABCD", "ABCDEFGH", false},
		{"empty", "", "ABCDEFGH", false},
		{"both empty", "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := authkit.EqualToken(tt.got, tt.want); got != tt.equal {
				t.Errorf("EqualToken(%q, %q) = %v, want %v", tt.got, tt.want, got, tt.equal)
			}
		})
	}
}

// ANCHOR_END: token

// ANCHOR: client

func TestClientTrustsOwnCA(t *testing.T) {
	cert := selfSigned(t)
	srv := startServer(t, cert)
	client, err := authkit.NewClient(caPEM(cert))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("trusted server: %v", err)
	}
	defer resp.Body.Close()
	if body, _ := io.ReadAll(resp.Body); string(body) != "ok" {
		t.Errorf("body = %q, want ok", body)
	}
}

func TestClientRejectsImpostor(t *testing.T) {
	cert := selfSigned(t)
	client, err := authkit.NewClient(caPEM(cert))
	if err != nil {
		t.Fatal(err)
	}
	impostor := startServer(t, selfSigned(t)) // 同樣的名稱與位址，但是另一把私鑰
	_, err = client.Get(impostor.URL)
	if _, ok := errors.AsType[x509.UnknownAuthorityError](err); !ok {
		t.Fatalf("err = %v, want x509.UnknownAuthorityError", err)
	}
}

func TestNewClientRejectsBadCA(t *testing.T) {
	if _, err := authkit.NewClient([]byte("not a pem")); err == nil {
		t.Fatal("want error for input without a certificate")
	}
}

// ANCHOR_END: client

// ANCHOR: version

func TestServerRequiresTLS13(t *testing.T) {
	cert := selfSigned(t)
	srv := startServer(t, cert)
	client, err := authkit.NewClient(caPEM(cert))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Get(srv.URL); err != nil {
		t.Fatalf("TLS 1.3 client: %v", err)
	}

	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(caPEM(cert))
	_, err = tls.Dial("tcp", srv.Listener.Addr().String(), &tls.Config{RootCAs: pool, MaxVersion: tls.VersionTLS12})
	// 伺服器送回 protocol_version alert，client 端看到的是 remote error。
	if err == nil || !strings.Contains(err.Error(), "protocol version not supported") {
		t.Fatalf("TLS 1.2 client: err = %v, want protocol version not supported", err)
	}
}

// ANCHOR_END: version
