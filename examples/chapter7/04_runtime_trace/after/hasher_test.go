package passwd_test

import (
	"crypto/pbkdf2"
	"crypto/sha256"
	"fmt"
	"slices"
	"strconv"
	"sync"
	"testing"

	passwd "github.com/holmeslin/learning-go-to-senior/examples/chapter7/04_runtime_trace/after"
)

// ANCHOR: example

func ExampleHasher_Hash() {
	h := passwd.New(1000)

	key, err := h.Hash("correct horse", []byte("example-salt"))
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("%x\n", key)
	// Output:
	// 8d7302a8f0252df7bd7334806adaca5bc13708c0930816c254aa80246635d009
}

// ANCHOR_END: example

// ANCHOR: concurrent

// 同時有很多人登入：每個人拿回的，必須是自己那組密碼與鹽的結果。
func TestHashConcurrentMatchesPBKDF2(t *testing.T) {
	const callers, iter = 32, 1000
	h := passwd.New(iter)
	salt := []byte("test-salt")

	got := make([][]byte, callers)
	var wg sync.WaitGroup
	for i := range callers {
		wg.Go(func() {
			key, err := h.Hash("password-"+strconv.Itoa(i), salt)
			if err != nil {
				t.Errorf("Hash(%d) error = %v", i, err)
			}
			got[i] = key
		})
	}
	wg.Wait()

	for i := range callers {
		want, err := pbkdf2.Key(sha256.New, "password-"+strconv.Itoa(i), salt, iter, passwd.KeyLen)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(got[i], want) {
			t.Errorf("Hash(%d) = %x, want %x", i, got[i], want)
		}
	}
}

// ANCHOR_END: concurrent
