package size_test

import (
	"errors"
	"fmt"
	"strconv"
	"testing"

	size "github.com/holmeslin/learning-go-to-senior/examples/chapter1/08_deprecation_and_compatibility/after"
)

var tests = []struct {
	in      string
	want    int64
	wantErr bool
}{
	{"512B", 512, false},
	{"1KiB", 1 << 10, false},
	{"10 MiB", 10 << 20, false},
	{"2GiB", 2 << 30, false},
	{"10", 0, true},
	{"MiB", 0, true},
	{"1TB", 0, true},
	{"9223372036854775807GiB", 0, true},
	{"99999999999999999999B", 0, true}, // 數字本身超出 int64
}

func TestParse(t *testing.T) {
	for _, tt := range tests {
		got, err := size.Parse(tt.in)
		if got != tt.want || (err != nil) != tt.wantErr {
			t.Errorf("Parse(%q) = %d, %v; want %d, error=%v", tt.in, got, err, tt.want, tt.wantErr)
		}
	}
}

// 舊名稱要和新名稱行為一致，錯誤的情況也一樣：結果、錯誤訊息與錯誤鏈都要相同，
// 舊呼叫端不改程式也不會察覺差異。
func TestParseSizeMatchesParse(t *testing.T) {
	for _, tt := range tests {
		want, wantErr := size.Parse(tt.in)
		got, gotErr := size.ParseSize(tt.in) // 舊呼叫端的寫法
		if got != want {
			t.Errorf("ParseSize(%q) = %d; Parse = %d", tt.in, got, want)
		}
		if fmt.Sprint(gotErr) != fmt.Sprint(wantErr) {
			t.Errorf("ParseSize(%q) error = %v; Parse error = %v", tt.in, gotErr, wantErr)
		}
		if errors.Is(gotErr, strconv.ErrRange) != errors.Is(wantErr, strconv.ErrRange) {
			t.Errorf("ParseSize(%q): errors.Is(ErrRange) differs from Parse", tt.in)
		}
	}
}
