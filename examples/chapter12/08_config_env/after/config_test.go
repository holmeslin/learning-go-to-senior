package relay_test

import (
	"errors"
	"flag"
	"io"
	"strings"
	"testing"
	"time"

	relay "github.com/holmeslin/learning-go-to-senior/examples/chapter12/08_config_env/after"
)

// ANCHOR: load

// envNames 是 Load 會讀的全部環境變數。
var envNames = []string{"RELAY_ADDR", "RELAY_UPSTREAM", "RELAY_TIMEOUT", "RELAY_MAX_RETRIES", "RELAY_DEBUG"}

// load 先把所有變數設成空字串，再套上 env 與 args。t.Setenv 讓測試結束時還原，
// 空字串等於沒設，所以結果不受開發機的環境影響。每次都用自己的 FlagSet。
func load(t *testing.T, env map[string]string, args ...string) (relay.Config, error) {
	t.Helper()
	for _, name := range envNames {
		t.Setenv(name, env[name])
	}
	fs := flag.NewFlagSet("relay", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return relay.Load(fs, args)
}

// ANCHOR_END: load

// ANCHOR: precedence

func TestPrecedence(t *testing.T) {
	const up = "http://hooks.internal:9000"
	defaults := relay.Config{Addr: ":8080", Upstream: up, Timeout: 5 * time.Second, MaxRetries: 3}
	with := func(f func(*relay.Config)) relay.Config { c := defaults; f(&c); return c }

	tests := []struct {
		name string
		env  map[string]string
		args []string
		want relay.Config
	}{
		{"defaults", map[string]string{"RELAY_UPSTREAM": up}, nil, defaults},
		{"env over default",
			map[string]string{"RELAY_UPSTREAM": up, "RELAY_TIMEOUT": "10s", "RELAY_DEBUG": "true"}, nil,
			with(func(c *relay.Config) { c.Timeout, c.Debug = 10*time.Second, true })},
		{"flag over env",
			map[string]string{"RELAY_UPSTREAM": up, "RELAY_TIMEOUT": "10s", "RELAY_MAX_RETRIES": "7"},
			[]string{"-timeout=2s", "-debug"},
			with(func(c *relay.Config) { c.Timeout, c.MaxRetries, c.Debug = 2*time.Second, 7, true })},
		{"flag wins even when env is garbage",
			map[string]string{"RELAY_UPSTREAM": up, "RELAY_TIMEOUT": "30"}, []string{"-timeout=1m"},
			with(func(c *relay.Config) { c.Timeout = time.Minute })},
		{"flag can switch a boolean off again",
			map[string]string{"RELAY_UPSTREAM": up, "RELAY_DEBUG": "true"}, []string{"-debug=false"}, defaults},
		{"empty env is unset", map[string]string{"RELAY_UPSTREAM": up, "RELAY_ADDR": ""}, nil, defaults},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := load(t, tt.env, tt.args...)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

// ANCHOR_END: precedence

// ANCHOR: errors

func TestAllErrorsAreReportedAtOnce(t *testing.T) {
	_, err := load(t, map[string]string{
		"RELAY_ADDR": "8080", "RELAY_TIMEOUT": "30", "RELAY_MAX_RETRIES": "three", "RELAY_DEBUG": "maybe",
	})
	want := []string{
		`addr (env RELAY_ADDR): address 8080: missing port in address`,
		`upstream: required; set -upstream or RELAY_UPSTREAM`,
		`timeout (env RELAY_TIMEOUT): time: missing unit in duration "30"`,
		`max-retries (env RELAY_MAX_RETRIES): strconv.Atoi: parsing "three": invalid syntax`,
		`debug (env RELAY_DEBUG): strconv.ParseBool: parsing "maybe": invalid syntax`,
	}
	if err == nil {
		t.Fatal("want an error")
	}
	if got := strings.Split(err.Error(), "\n"); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("got:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestErrorNamesTheSource(t *testing.T) {
	_, err := load(t, map[string]string{"RELAY_UPSTREAM": "http://a", "RELAY_TIMEOUT": "10s"}, "-timeout=abc")
	if want := `timeout (flag -timeout): time: invalid duration "abc"`; err == nil || err.Error() != want {
		t.Errorf("got %v, want %s", err, want)
	}
}

// ANCHOR_END: errors

// ANCHOR: validate

func TestValidation(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want string
	}{
		{"zero timeout", map[string]string{"RELAY_TIMEOUT": "0s"}, "timeout (env RELAY_TIMEOUT): must be positive, got 0s"},
		{"negative timeout", map[string]string{"RELAY_TIMEOUT": "-1s"}, "timeout (env RELAY_TIMEOUT): must be positive, got -1s"},
		{"too many retries", map[string]string{"RELAY_MAX_RETRIES": "11"}, "max-retries (env RELAY_MAX_RETRIES): must be between 0 and 10, got 11"},
		{"negative retries", map[string]string{"RELAY_MAX_RETRIES": "-1"}, "max-retries (env RELAY_MAX_RETRIES): must be between 0 and 10, got -1"},
		{"port out of range", map[string]string{"RELAY_ADDR": ":70000"}, "addr (env RELAY_ADDR): port 70000 is out of range 0-65535"},
		{"port not a number", map[string]string{"RELAY_ADDR": ":80a80"}, `addr (env RELAY_ADDR): port "80a80" is not a number`},
		{"upstream scheme", map[string]string{"RELAY_UPSTREAM": "ftp://hooks.internal"}, `upstream (env RELAY_UPSTREAM): want an http or https url with a host`},
		{"upstream without host", map[string]string{"RELAY_UPSTREAM": "hooks.internal"}, `upstream (env RELAY_UPSTREAM): want an http or https url with a host`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := map[string]string{"RELAY_UPSTREAM": "http://hooks.internal"}
			for k, v := range tt.env {
				env[k] = v
			}
			_, err := load(t, env)
			if err == nil || err.Error() != tt.want {
				t.Errorf("got %v, want %s", err, tt.want)
			}
		})
	}
}

// ANCHOR_END: validate

func TestFlagSyntaxErrors(t *testing.T) {
	env := map[string]string{"RELAY_UPSTREAM": "http://hooks.internal"}
	if _, err := load(t, env, "-nope"); err == nil || !strings.Contains(err.Error(), "flag provided but not defined: -nope") {
		t.Errorf("unknown flag: got %v", err)
	}
	if _, err := load(t, env, "-h"); !errors.Is(err, flag.ErrHelp) {
		t.Errorf("-h: got %v, want flag.ErrHelp", err)
	}
	if _, err := load(t, env, "extra"); err == nil || err.Error() != `unexpected arguments: ["extra"]` {
		t.Errorf("positional argument: got %v", err)
	}
}

func TestNewClientUsesTheConfig(t *testing.T) {
	cfg, err := load(t, map[string]string{"RELAY_UPSTREAM": "http://hooks.internal", "RELAY_TIMEOUT": "3s"})
	if err != nil {
		t.Fatal(err)
	}
	// 換掉環境變數，已經建好的設定不受影響：NewClient 只看 Config。
	t.Setenv("RELAY_TIMEOUT", "1h")
	c := relay.NewClient(cfg)
	if c.Timeout != 3*time.Second {
		t.Errorf("Timeout = %v, want 3s", c.Timeout)
	}
}
