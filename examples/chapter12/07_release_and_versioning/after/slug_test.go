package slugtool_test

import (
	"fmt"
	"testing"

	slugtool "github.com/holmeslin/learning-go-to-senior/examples/chapter12/07_release_and_versioning/after"
)

// v2 的 Slug 多回傳 error：空白的標題不再悄悄得到空字串。
func Example() {
	s, err := slugtool.Slug("Hello Go World")
	fmt.Printf("%q %v\n", s, err)
	_, err = slugtool.Slug("   ")
	fmt.Println(err)
	// Output:
	// "hello-go-world" <nil>
	// slugtool: title has no usable characters
}

func TestSlug(t *testing.T) {
	tests := []struct {
		title   string
		want    string
		wantErr bool
	}{
		{"Hello Go World", "hello-go-world", false},
		{"  Go   Modules ", "go-modules", false},
		{"", "", true},
		{"\t\n", "", true},
	}
	for _, tt := range tests {
		got, err := slugtool.Slug(tt.title)
		if got != tt.want || (err != nil) != tt.wantErr {
			t.Errorf("Slug(%q) = %q, %v; want %q, error=%v", tt.title, got, err, tt.want, tt.wantErr)
		}
	}
}
