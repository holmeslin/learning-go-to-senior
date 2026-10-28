// Package server 是伺服器的啟動流程。
package server

import (
	"fmt"
	"io/fs"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter2/04_error_message_style/after/config"
)

// Server 是設定載入完成的伺服器。
type Server struct {
	// Addr 是監聽位址，例如 ":8080"。
	Addr string
}

// ANCHOR: new

// New 讀取設定並建立 Server。
func New(fsys fs.FS, name string) (*Server, error) {
	port, err := config.Load(fsys, name)
	if err != nil {
		return nil, fmt.Errorf("create server: %w", err)
	}
	return &Server{Addr: fmt.Sprintf(":%d", port)}, nil
}

// ANCHOR_END: new
