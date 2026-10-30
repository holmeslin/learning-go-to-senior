package reports_test

import (
	"errors"
	"fmt"
	"os"
	"strings"

	reports "github.com/holmeslin/learning-go-to-senior/examples/chapter11/03_path_from_user_input/after"
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
	store, err := reports.Open(dir)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer store.Close()

	for _, a := range attacks {
		data, err := store.Read(a.name)
		switch {
		case errors.Is(err, reports.ErrInvalidName):
			fmt.Printf("%s：檔名不合法\n", a.label)
		case err != nil:
			fmt.Printf("%s：os.Root 拒絕\n", a.label)
		case strings.Contains(string(data), secret):
			fmt.Printf("%s：讀到目錄外的檔案\n", a.label)
		default:
			fmt.Printf("%s：讀到\n", a.label)
		}
	}
	// Output:
	// 正常檔名：讀到
	// 子目錄：讀到
	// ..：檔名不合法
	// .. 加相同開頭的目錄：檔名不合法
	// 絕對路徑：檔名不合法
	// 符號連結：os.Root 拒絕
}

// ANCHOR_END: example
