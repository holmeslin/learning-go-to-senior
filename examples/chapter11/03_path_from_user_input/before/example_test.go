package reports_test

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	reports "github.com/holmeslin/learning-go-to-senior/examples/chapter11/03_path_from_user_input/before"
)

// ANCHOR: example

func ExampleStore_Read() {
	base, err := os.MkdirTemp("", "reports-*")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer os.RemoveAll(base)

	dir, attacks, err := setup(base)
	if err != nil {
		fmt.Println(err)
		return
	}
	store := &reports.Store{Dir: dir}

	for _, a := range attacks {
		data, err := store.Read(a.name)
		switch {
		case errors.Is(err, reports.ErrOutside):
			fmt.Printf("%s：檢查擋下\n", a.label)
		case errors.Is(err, fs.ErrNotExist):
			fmt.Printf("%s：找不到檔案\n", a.label)
		case err != nil:
			fmt.Printf("%s：%v\n", a.label, err)
		case strings.Contains(string(data), secret):
			fmt.Printf("%s：讀到目錄外的檔案\n", a.label)
		default:
			fmt.Printf("%s：讀到\n", a.label)
		}
	}
	// Output:
	// 正常檔名：讀到
	// 子目錄：讀到
	// ..：檢查擋下
	// .. 加相同開頭的目錄：讀到目錄外的檔案
	// 絕對路徑：找不到檔案
	// 符號連結：讀到目錄外的檔案
}

// ANCHOR_END: example
