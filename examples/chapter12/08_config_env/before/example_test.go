package relay_test

import (
	"fmt"
	"os"

	relay "github.com/holmeslin/learning-go-to-senior/examples/chapter12/08_config_env/before"
)

// ANCHOR: setenv

// setEnv 設定一組環境變數，回傳把它們還原成原狀的函式，原本沒設的就還原成沒設。
func setEnv(vars map[string]string) (restore func()) {
	old := make(map[string]*string, len(vars))
	for name, value := range vars {
		if prev, ok := os.LookupEnv(name); ok {
			old[name] = &prev
		} else {
			old[name] = nil
		}
		os.Setenv(name, value)
	}
	return func() {
		for name, prev := range old {
			if prev == nil {
				os.Unsetenv(name)
			} else {
				os.Setenv(name, *prev)
			}
		}
	}
}

// ANCHOR_END: setenv

// ANCHOR: example

// 維運的人設了四個變數：逾時寫成 30s、重試次數打錯字、debug 想關掉、上游忘了設。
// 沒有任何一個被回報，服務照樣啟動。
func Example() {
	defer setEnv(map[string]string{
		"RELAY_PORT": "", "RELAY_UPSTREAM": "", "RELAY_TIMEOUT_SEC": "30s",
		"RELAY_MAX_RETRIES": "three", "RELAY_DEBUG": "false",
	})()

	fmt.Println("addr:", relay.Addr())
	fmt.Println("upstream:", relay.Upstream())
	fmt.Println("client timeout:", relay.NewClient().Timeout)
	fmt.Println("max retries:", relay.MaxRetries())
	fmt.Println("debug:", relay.Debug())
	// Output:
	// addr: :0
	// upstream: http://localhost:8080
	// client timeout: 0s
	// max retries: 0
	// debug: true
}

// ANCHOR_END: example
