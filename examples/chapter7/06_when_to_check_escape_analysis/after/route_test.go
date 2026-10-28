package accesslog_test

import (
	"fmt"
	"testing"

	after "github.com/holmeslin/learning-go-to-senior/examples/chapter7/06_when_to_check_escape_analysis/after"
	before "github.com/holmeslin/learning-go-to-senior/examples/chapter7/06_when_to_check_escape_analysis/before"
)

var lines = []string{
	"GET /users/42 200",
	"POST /orders 201",
	"GET /healthz 503",
	"DELETE /users/42 404",
}

var badLines = []string{
	"",
	"GET",
	"GET /users/42",
	"GET /users/42 abc",
	"GET /users/42 99",
}

// ANCHOR: same

// 改寫前後必須得到一樣的結果，錯誤訊息也一樣。
func TestRouteSameAsBefore(t *testing.T) {
	for _, line := range append(lines, badLines...) {
		want, wantErr := before.Route(line)
		got, gotErr := after.Route(line)
		if got != want || fmt.Sprint(gotErr) != fmt.Sprint(wantErr) {
			t.Errorf("Route(%q) = %q, %v; before = %q, %v", line, got, gotErr, want, wantErr)
		}

		// 出錯時 AppendRoute 原樣回傳 dst，不留下半截結果。
		gotBuf, gotErr := after.AppendRoute([]byte("> "), line)
		if wantErr == nil {
			want = "> " + want
		} else {
			want = "> "
		}
		if string(gotBuf) != want || fmt.Sprint(gotErr) != fmt.Sprint(wantErr) {
			t.Errorf("AppendRoute(%q) = %q, %v; want %q, %v", line, gotBuf, gotErr, want, wantErr)
		}
	}
}

// ANCHOR_END: same

// ANCHOR: allocs

// 呼叫端重複使用緩衝區時，熱點函式不配置。
func TestAppendRouteDoesNotAllocate(t *testing.T) {
	buf := make([]byte, 0, 64)
	for _, line := range lines {
		allocs := testing.AllocsPerRun(100, func() {
			buf, _ = after.AppendRoute(buf[:0], line)
		})
		if allocs != 0 {
			t.Errorf("AppendRoute(%q) allocs = %v, want 0", line, allocs)
		}
	}
}

// ANCHOR_END: allocs

func BenchmarkRoute(b *testing.B) {
	buf := make([]byte, 0, 64)
	for b.Loop() {
		for _, line := range lines {
			buf, _ = after.AppendRoute(buf[:0], line)
		}
	}
}

// ANCHOR: example

func ExampleAppendRoute() {
	buf, err := after.AppendRoute(nil, "GET /users/42 503")
	fmt.Println(string(buf), err)

	_, err = after.AppendRoute(nil, "GET /users/42 99")
	fmt.Println(err)
	// Output:
	// GET /users/42 5xx <nil>
	// bad status "99"
}

// ANCHOR_END: example
