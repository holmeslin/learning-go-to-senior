package relay

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

// Config 是轉發服務的設定。Load 驗證過之後才會交出來，
// 拿到 Config 的程式不必再檢查欄位，也不該再讀環境變數。
type Config struct {
	// Addr 是服務監聽的位址，格式是 host:port。
	Addr string
	// Upstream 是轉發的目的地，必須是帶主機的 http 或 https 網址。
	Upstream string
	// Timeout 是呼叫上游的逾時，必須大於零。
	Timeout time.Duration
	// MaxRetries 是呼叫上游失敗後最多重試幾次，範圍是 0 到 10。
	MaxRetries int
	// Debug 打開時輸出除錯日誌。
	Debug bool
}

// ANCHOR: load

// Load 在 fs 上註冊旗標、解析 args，再依「旗標、環境變數、預設值」的順序
// 決定每個欄位的值。環境變數設成空字串視為沒設。fs 由呼叫端建立，
// 同一個 fs 不能呼叫兩次 Load，否則旗標會重複註冊而 panic。
//
// 旗標本身的語法錯誤（未定義的旗標、-h 的 flag.ErrHelp）在解析時就回傳；
// 值的轉換與驗證錯誤則全部收集起來，用 errors.Join 一次回傳。
// upstream 沒有預設值，三個來源都沒給就是錯誤。
func Load(fs *flag.FlagSet, args []string) (Config, error) {
	// 每個旗標只記下使用者給的原始字串，轉換與驗證集中在下面。
	given := map[string]string{}
	str := func(name, usage string) {
		fs.Func(name, usage, func(v string) error { given[name] = v; return nil })
	}
	str("addr", "listen address host:port (env RELAY_ADDR, default :8080)")
	str("upstream", "upstream url (env RELAY_UPSTREAM, required)")
	str("timeout", "upstream timeout, e.g. 5s (env RELAY_TIMEOUT, default 5s)")
	str("max-retries", "retries after a failed call, 0-10 (env RELAY_MAX_RETRIES, default 3)")
	fs.BoolFunc("debug", "enable debug logging (env RELAY_DEBUG, default false)",
		func(v string) error { given["debug"] = v; return nil })

	if err := fs.Parse(args); err != nil {
		return Config{}, fmt.Errorf("parse flags: %w", err)
	}
	if fs.NArg() > 0 {
		return Config{}, fmt.Errorf("unexpected arguments: %q", fs.Args())
	}

	var cfg Config
	var errs []error
	// field 取出欄位的值與來源；轉換失敗時，錯誤帶上名稱與來源。
	field := func(name, env, def string, convert func(string) error) {
		v, from := def, "default"
		if g, ok := given[name]; ok {
			v, from = g, "flag -"+name
		} else if e := os.Getenv(env); e != "" {
			v, from = e, "env "+env
		}
		if err := convert(v); err != nil {
			errs = append(errs, fmt.Errorf("%s (%s): %w", name, from, err))
		}
	}

	field("addr", "RELAY_ADDR", ":8080", func(v string) error { cfg.Addr = v; return checkAddr(v) })
	if _, ok := given["upstream"]; !ok && os.Getenv("RELAY_UPSTREAM") == "" {
		errs = append(errs, errors.New("upstream: required; set -upstream or RELAY_UPSTREAM"))
	} else {
		field("upstream", "RELAY_UPSTREAM", "", func(v string) error { cfg.Upstream = v; return checkUpstream(v) })
	}
	field("timeout", "RELAY_TIMEOUT", "5s", func(v string) (err error) {
		if cfg.Timeout, err = time.ParseDuration(v); err == nil && cfg.Timeout <= 0 {
			err = fmt.Errorf("must be positive, got %v", cfg.Timeout)
		}
		return err
	})
	field("max-retries", "RELAY_MAX_RETRIES", "3", func(v string) (err error) {
		if cfg.MaxRetries, err = strconv.Atoi(v); err == nil && (cfg.MaxRetries < 0 || cfg.MaxRetries > 10) {
			err = fmt.Errorf("must be between 0 and 10, got %d", cfg.MaxRetries)
		}
		return err
	})
	field("debug", "RELAY_DEBUG", "false", func(v string) (err error) { cfg.Debug, err = strconv.ParseBool(v); return err })

	if err := errors.Join(errs...); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// ANCHOR_END: load

// checkAddr 檢查 host:port，port 要是 0 到 65535 的數字。
func checkAddr(v string) error {
	_, port, err := net.SplitHostPort(v)
	if err != nil {
		return err
	}
	p, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("port %q is not a number", port)
	}
	if p < 0 || p > 65535 {
		return fmt.Errorf("port %d is out of range 0-65535", p)
	}
	return nil
}

// checkUpstream 檢查網址。錯誤不帶網址的內容：網址可能夾帶帳號密碼。
func checkUpstream(v string) error {
	u, err := url.Parse(v)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return errors.New("want an http or https url with a host")
	}
	return nil
}

// ANCHOR: client

// NewClient 用設定建立呼叫上游的 HTTP client。設定是參數傳進來的，
// 這裡與呼叫它的程式都不讀環境變數。
func NewClient(cfg Config) *http.Client {
	return &http.Client{Timeout: cfg.Timeout}
}

// ANCHOR_END: client
