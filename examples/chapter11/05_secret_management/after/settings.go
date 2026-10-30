package settings

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// Config 是報表服務的設定。秘密欄位的型別是 Secret，其餘是一般字串。
type Config struct {
	// DBHost 是資料庫的位址。
	DBHost string
	// DBUser 是連資料庫的帳號。
	DBUser string
	// DBPassword 是連資料庫的密碼。
	DBPassword Secret
	// APIKey 是呼叫第三方報表 API 的金鑰。
	APIKey Secret
}

// ANCHOR: load

// Load 從環境變數讀出秘密。REPORT_API_KEY_FILE 有設就改讀那個檔案
// （容器平台把秘密掛成檔案的做法），檔案結尾的換行會去掉。
// 任何一個秘密讀不到或是空的都回傳錯誤，呼叫端應該在啟動時就結束；
// 錯誤只寫變數名稱，不寫值。
func Load() (Config, error) {
	apiKey, apiErr := readSecret("REPORT_API_KEY")
	password, pwErr := readSecret("REPORT_DB_PASSWORD")
	if err := errors.Join(apiErr, pwErr); err != nil {
		return Config{}, err
	}
	// 位址與帳號不是秘密；它們怎麼載入，第 12 章〈設定與環境變數〉會談。
	return Config{DBHost: "db.internal:5432", DBUser: "report", DBPassword: password, APIKey: apiKey}, nil
}

func readSecret(name string) (Secret, error) {
	value := os.Getenv(name)
	if path := os.Getenv(name + "_FILE"); path != "" {
		b, err := os.ReadFile(path)
		if err != nil {
			return Secret{}, fmt.Errorf("read %s_FILE: %w", name, err)
		}
		value = strings.TrimRight(string(b), "\r\n")
	}
	if value == "" {
		return Secret{}, fmt.Errorf("%s is empty or not set; set it or point %s_FILE at a file", name, name)
	}
	return Secret{value: value}, nil
}

// ANCHOR_END: load

// DSN 回傳連線字串。裡面有密碼，所以它本身也是 Secret。
func (c Config) DSN() Secret {
	return Secret{value: fmt.Sprintf("postgres://%s:%s@%s/reports", c.DBUser, c.DBPassword.Reveal(), c.DBHost)}
}

// ANCHOR: connect

// Connect 用 open 開資料庫連線；open 是真正連線的函式，例如包著 sql.Open 與 Ping。
// 失敗的錯誤只帶位址，不帶連線字串。
func Connect(c Config, open func(dsn string) error) error {
	if err := open(c.DSN().Reveal()); err != nil {
		return fmt.Errorf("connect to %s: %w", c.DBHost, err)
	}
	return nil
}

// ANCHOR_END: connect
