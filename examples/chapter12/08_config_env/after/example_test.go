package relay_test

import (
	"flag"
	"fmt"
	"io"
	"os"

	relay "github.com/holmeslin/learning-go-to-senior/examples/chapter12/08_config_env/after"
)

// ANCHOR: setenv

// setEnv 設定一組環境變數，回傳把它們還原成原狀的函式，原本沒設的就還原成沒設。
// Example 沒有 *testing.T，不能用 t.Setenv，所以自己存、自己還原。
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

// 進入點要做的事：載入、失敗就結束、把設定往下傳。環境變數給了逾時與上游，
// 命令列用旗標把逾時改成 2s、打開 debug。
func Example() {
	defer setEnv(map[string]string{
		"RELAY_ADDR": "", "RELAY_UPSTREAM": "http://hooks.internal:9000", "RELAY_TIMEOUT": "10s",
		"RELAY_MAX_RETRIES": "", "RELAY_DEBUG": "",
	})()

	fs := flag.NewFlagSet("relay", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	cfg, err := relay.Load(fs, []string{"-timeout=2s", "-debug"})
	if err != nil {
		fmt.Println(err)
		return
	}
	client := relay.NewClient(cfg)
	fmt.Printf("%+v\n", cfg)
	fmt.Println("client timeout:", client.Timeout)
	// Output:
	// {Addr::8080 Upstream:http://hooks.internal:9000 Timeout:2s MaxRetries:3 Debug:true}
	// client timeout: 2s
}

// ANCHOR_END: example

// ANCHOR: bad

// 同樣的錯誤輸入（before 悄悄吞掉的那幾個），這次一次全部報出來。
func ExampleLoad_errors() {
	defer setEnv(map[string]string{
		"RELAY_ADDR": "8080", "RELAY_UPSTREAM": "", "RELAY_TIMEOUT": "30",
		"RELAY_MAX_RETRIES": "three", "RELAY_DEBUG": "maybe",
	})()

	fs := flag.NewFlagSet("relay", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	_, err := relay.Load(fs, nil)
	fmt.Println(err)
	// Output:
	// addr (env RELAY_ADDR): address 8080: missing port in address
	// upstream: required; set -upstream or RELAY_UPSTREAM
	// timeout (env RELAY_TIMEOUT): time: missing unit in duration "30"
	// max-retries (env RELAY_MAX_RETRIES): strconv.Atoi: parsing "three": invalid syntax
	// debug (env RELAY_DEBUG): strconv.ParseBool: parsing "maybe": invalid syntax
}

// ANCHOR_END: bad
