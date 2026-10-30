package settings_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	settings "github.com/holmeslin/learning-go-to-senior/examples/chapter11/05_secret_management/after"
)

// ANCHOR: example

// 同樣的三條外洩路徑：%v 印設定、連線失敗的錯誤訊息、JSON 輸出。
func Example() {
	os.Setenv("REPORT_API_KEY", "example-not-a-real-key")
	os.Setenv("REPORT_API_KEY_FILE", "")
	os.Setenv("REPORT_DB_PASSWORD", "example-not-a-real-password")
	os.Setenv("REPORT_DB_PASSWORD_FILE", "")
	cfg, err := settings.Load()
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("%v\n", cfg)
	fmt.Printf("%#v\n", cfg)

	err = settings.Connect(cfg, func(string) error { return errors.New("connection refused") })
	fmt.Println(err)

	b, _ := json.Marshal(cfg)
	fmt.Println(string(b))
	// Output:
	// {db.internal:5432 report [REDACTED] [REDACTED]}
	// settings.Config{DBHost:"db.internal:5432", DBUser:"report", DBPassword:[REDACTED], APIKey:[REDACTED]}
	// connect to db.internal:5432: connection refused
	// {"DBHost":"db.internal:5432","DBUser":"report","DBPassword":"[REDACTED]","APIKey":"[REDACTED]"}
}

// ANCHOR_END: example

// ANCHOR: missing

// 讀不到秘密就啟動失敗，錯誤只說缺哪個變數。
func ExampleLoad_missing() {
	for _, name := range []string{"REPORT_API_KEY", "REPORT_API_KEY_FILE", "REPORT_DB_PASSWORD", "REPORT_DB_PASSWORD_FILE"} {
		os.Setenv(name, "")
	}
	_, err := settings.Load()
	fmt.Println(err)
	// Output:
	// REPORT_API_KEY is empty or not set; set it or point REPORT_API_KEY_FILE at a file
	// REPORT_DB_PASSWORD is empty or not set; set it or point REPORT_DB_PASSWORD_FILE at a file
}

// ANCHOR_END: missing
