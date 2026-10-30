// Package settings 是刻意示範的漏洞，不要照用：金鑰寫死在原始碼裡，
// 設定 struct 的秘密是一般字串，印出來、序列化、放進錯誤訊息時都會原樣露出。
// 可以參考的寫法在 after/。
package settings

import "fmt"

// ANCHOR: config

// apiKey 寫死在程式碼裡，會跟著版本控制的歷史一起保存。
const apiKey = "example-not-a-real-key"

// Config 是報表服務的設定。
type Config struct {
	// DBHost 是資料庫的位址。
	DBHost string
	// DBUser 是連資料庫的帳號。
	DBUser string
	// DBPassword 是連資料庫的密碼。
	DBPassword string
	// APIKey 是呼叫第三方報表 API 的金鑰。
	APIKey string
}

// Load 回傳設定，密碼與金鑰都寫死。
func Load() Config {
	return Config{DBHost: "db.internal:5432", DBUser: "report", DBPassword: "example-not-a-real-password", APIKey: apiKey}
}

// ANCHOR_END: config

// DSN 回傳連線字串，裡面有密碼。
func (c Config) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s/reports", c.DBUser, c.DBPassword, c.DBHost)
}

// ANCHOR: connect

// Connect 用 open 開資料庫連線；失敗時把連線字串寫進錯誤訊息。
func Connect(c Config, open func(dsn string) error) error {
	if err := open(c.DSN()); err != nil {
		return fmt.Errorf("open database %s: %w", c.DSN(), err)
	}
	return nil
}

// ANCHOR_END: connect
