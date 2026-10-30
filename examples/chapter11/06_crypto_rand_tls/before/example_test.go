package authkit_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	authkit "github.com/holmeslin/learning-go-to-senior/examples/chapter11/06_crypto_rand_tls/before"
)

// ANCHOR: example

func Example() {
	// httptest 的憑證不是任何 CA 簽的：對這個 client 來說，它就是冒充的伺服器。
	impostor := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, "balance: 0")
	}))
	defer impostor.Close()

	resp, err := authkit.NewClient().Get(impostor.URL)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("%s (%s)\n", body, resp.Status)
	// Output: balance: 0 (200 OK)
}

// ANCHOR_END: example
