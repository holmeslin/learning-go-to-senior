package server_test

import (
	"errors"
	"fmt"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter2/04_error_message_style/after/server"
)

// ANCHOR: example

func ExampleNew() {
	missing := fstest.MapFS{}
	badPort := fstest.MapFS{"app.conf": {Data: []byte("abc")}}

	_, err := server.New(missing, "app.conf")
	fmt.Println(err)
	_, err = server.New(badPort, "app.conf")
	fmt.Println(err)
	// Output:
	// create server: load config: open app.conf: file does not exist
	// create server: load config: parse port: strconv.Atoi: parsing "abc": invalid syntax
}

// ANCHOR_END: example

// ANCHOR: user

// userMessage 是邊界（例如 main）才做的事：把錯誤翻成使用者看得懂的一句話。
func userMessage(err error) string {
	if errors.Is(err, fs.ErrNotExist) {
		return "找不到設定檔 app.conf，請確認檔案存在"
	}
	return "伺服器無法啟動，詳細原因請看日誌"
}

func ExampleNew_userMessage() {
	_, err := server.New(fstest.MapFS{}, "app.conf")
	fmt.Println("使用者看到：", userMessage(err))
	fmt.Println("日誌裡寫著：", err)
	// Output:
	// 使用者看到： 找不到設定檔 app.conf，請確認檔案存在
	// 日誌裡寫著： create server: load config: open app.conf: file does not exist
}

// ANCHOR_END: user

func TestNewKeepsErrorChain(t *testing.T) {
	_, err := server.New(fstest.MapFS{}, "app.conf")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("errors.Is(err, fs.ErrNotExist) = false, err = %v", err)
	}
}

func TestNewOK(t *testing.T) {
	s, err := server.New(fstest.MapFS{"app.conf": {Data: []byte("8080\n")}}, "app.conf")
	if err != nil {
		t.Fatal(err)
	}
	if s.Addr != ":8080" {
		t.Errorf("Addr = %q, want :8080", s.Addr)
	}
}
