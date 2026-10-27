package server_test

import (
	"fmt"
	"testing/fstest"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter2/04_error_message_style/before/server"
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
	// Failed to create server: Failed to load config: Error: failed to open file app.conf: open app.conf: file does not exist
	// Failed to create server: Failed to load config: Error: failed to parse port: strconv.Atoi: parsing "abc": invalid syntax
}

// ANCHOR_END: example
