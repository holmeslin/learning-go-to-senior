package reports_test

import (
	"errors"
	"os"
	"strings"
	"testing"

	reports "github.com/holmeslin/learning-go-to-senior/examples/chapter11/03_path_from_user_input/after"
)

// ANCHOR: test

func TestRead(t *testing.T) {
	dir, attacks, err := setup(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	store, err := reports.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	tests := map[string]struct {
		wantErr     bool // 是否被拒絕
		wantInvalid bool // 被拒絕時，是不是在檢查檔名那一關
	}{
		"正常檔名":        {},
		"子目錄":         {},
		"..":          {wantErr: true, wantInvalid: true},
		".. 加相同開頭的目錄": {wantErr: true, wantInvalid: true},
		"絕對路徑":        {wantErr: true, wantInvalid: true},
		"符號連結":        {wantErr: true}, // 檔名看不出問題，由 os.Root 擋下
	}
	for _, a := range attacks {
		want := tests[a.label]
		data, err := store.Read(a.name)
		if (err != nil) != want.wantErr {
			t.Errorf("%s: err = %v, wantErr %v", a.label, err, want.wantErr)
		}
		if got := errors.Is(err, reports.ErrInvalidName); got != want.wantInvalid {
			t.Errorf("%s: ErrInvalidName = %v, want %v", a.label, got, want.wantInvalid)
		}
		if strings.Contains(string(data), secret) {
			t.Errorf("%s: 讀到了報表目錄之外的內容", a.label)
		}
	}
}

// ANCHOR_END: test

// os.Root 只管不出目錄；目錄裡的 .state 它照樣讀得到，要不要給是檔名檢查的事。
func TestNameCheckIsSeparateFromRoot(t *testing.T) {
	dir, _, err := setup(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	store, err := reports.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if _, err := root.ReadFile(".state"); err != nil {
		t.Fatalf("os.Root 讀 .state: %v", err)
	}

	for _, name := range []string{".state", "SALES.CSV", "sales.csv/..", ""} {
		if _, err := store.Read(name); !errors.Is(err, reports.ErrInvalidName) {
			t.Errorf("Read(%q) err = %v, want ErrInvalidName", name, err)
		}
	}
}
