package settings_test

import (
	"encoding/json"
	"errors"
	"fmt"

	settings "github.com/holmeslin/learning-go-to-senior/examples/chapter11/05_secret_management/before"
)

// ANCHOR: example

// 三條外洩路徑：%v 印設定、連線失敗的錯誤訊息、JSON 輸出。
func Example() {
	cfg := settings.Load()
	fmt.Printf("%v\n", cfg)

	err := settings.Connect(cfg, func(string) error { return errors.New("connection refused") })
	fmt.Println(err)

	b, _ := json.Marshal(cfg)
	fmt.Println(string(b))
	// Output:
	// {db.internal:5432 report example-not-a-real-password example-not-a-real-key}
	// open database postgres://report:example-not-a-real-password@db.internal:5432/reports: connection refused
	// {"DBHost":"db.internal:5432","DBUser":"report","DBPassword":"example-not-a-real-password","APIKey":"example-not-a-real-key"}
}

// ANCHOR_END: example
